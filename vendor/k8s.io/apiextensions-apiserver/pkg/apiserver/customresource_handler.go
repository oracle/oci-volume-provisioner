/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/golang/glog"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/metrics"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/scale/scheme/autoscalingv1"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/conversion"
	apiservervalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	informers "k8s.io/apiextensions-apiserver/pkg/client/informers/internalversion/apiextensions/internalversion"
	listers "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/internalversion"
	"k8s.io/apiextensions-apiserver/pkg/controller/finalizer"
	apiextensionsfeatures "k8s.io/apiextensions-apiserver/pkg/features"
	"k8s.io/apiextensions-apiserver/pkg/registry/customresource"
	"k8s.io/apiextensions-apiserver/pkg/registry/customresource/tableconvertor"
)

// crdHandler serves the `/apis` endpoint.
// This is registered as a filter so that it never collides with any explicitly registered endpoints
type crdHandler struct {
	versionDiscoveryHandler *versionDiscoveryHandler
	groupDiscoveryHandler   *groupDiscoveryHandler

	customStorageLock sync.Mutex
	// customStorage contains a crdStorageMap
	// atomic.Value has a very good read performance compared to sync.RWMutex
	// see https://gist.github.com/dim/152e6bf80e1384ea72e17ac717a5000a
	// which is suited for most read and rarely write cases
	customStorage atomic.Value

	crdLister listers.CustomResourceDefinitionLister

	delegate          http.Handler
	restOptionsGetter generic.RESTOptionsGetter
	admission         admission.Interface
}

// crdInfo stores enough information to serve the storage for the custom resource
type crdInfo struct {
	// spec and acceptedNames are used to compare against if a change is made on a CRD. We only update
	// the storage if one of these changes.
	spec          *apiextensions.CustomResourceDefinitionSpec
	acceptedNames *apiextensions.CustomResourceDefinitionNames

	// Storage per version
	storages map[string]customresource.CustomResourceStorage

	// Request scope per version
	requestScopes map[string]handlers.RequestScope

	// Scale scope per version
	scaleRequestScopes map[string]handlers.RequestScope

	// Status scope per version
	statusRequestScopes map[string]handlers.RequestScope

	// storageVersion is the CRD version used when storing the object in etcd.
	storageVersion string
}

// crdStorageMap goes from customresourcedefinition to its storage
type crdStorageMap map[types.UID]*crdInfo

func NewCustomResourceDefinitionHandler(
	versionDiscoveryHandler *versionDiscoveryHandler,
	groupDiscoveryHandler *groupDiscoveryHandler,
	crdInformer informers.CustomResourceDefinitionInformer,
	delegate http.Handler,
	restOptionsGetter generic.RESTOptionsGetter,
	admission admission.Interface) *crdHandler {
	ret := &crdHandler{
		versionDiscoveryHandler: versionDiscoveryHandler,
		groupDiscoveryHandler:   groupDiscoveryHandler,
		customStorage:           atomic.Value{},
		crdLister:               crdInformer.Lister(),
		delegate:                delegate,
		restOptionsGetter:       restOptionsGetter,
		admission:               admission,
	}
	crdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: ret.updateCustomResourceDefinition,
		DeleteFunc: func(obj interface{}) {
			ret.removeDeadStorage()
		},
	})

	ret.customStorage.Store(crdStorageMap{})

	return ret
}

func (r *crdHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	requestInfo, ok := apirequest.RequestInfoFrom(ctx)
	if !ok {
		responsewriters.InternalError(w, req, fmt.Errorf("no RequestInfo found in the context"))
		return
	}
	if !requestInfo.IsResourceRequest {
		pathParts := splitPath(requestInfo.Path)
		// only match /apis/<group>/<version>
		// only registered under /apis
		if len(pathParts) == 3 {
			r.versionDiscoveryHandler.ServeHTTP(w, req)
			return
		}
		// only match /apis/<group>
		if len(pathParts) == 2 {
			r.groupDiscoveryHandler.ServeHTTP(w, req)
			return
		}

		r.delegate.ServeHTTP(w, req)
		return
	}

	crdName := requestInfo.Resource + "." + requestInfo.APIGroup
	crd, err := r.crdLister.Get(crdName)
	if apierrors.IsNotFound(err) {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !apiextensions.HasServedCRDVersion(crd, requestInfo.APIVersion) {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if !apiextensions.IsCRDConditionTrue(crd, apiextensions.Established) {
		r.delegate.ServeHTTP(w, req)
		return
	}

	terminating := apiextensions.IsCRDConditionTrue(crd, apiextensions.Terminating)

	crdInfo, err := r.getOrCreateServingInfoFor(crd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	verb := strings.ToUpper(requestInfo.Verb)
	resource := requestInfo.Resource
	subresource := requestInfo.Subresource
	scope := metrics.CleanScope(requestInfo)
	supportedTypes := []string{
		string(types.JSONPatchType),
		string(types.MergePatchType),
	}

	var handler http.HandlerFunc
	switch {
	case subresource == "status" && crd.Spec.Subresources != nil && crd.Spec.Subresources.Status != nil:
		handler = r.serveStatus(w, req, requestInfo, crdInfo, terminating, supportedTypes)
	case subresource == "scale" && crd.Spec.Subresources != nil && crd.Spec.Subresources.Scale != nil:
		handler = r.serveScale(w, req, requestInfo, crdInfo, terminating, supportedTypes)
	case len(subresource) == 0:
		handler = r.serveResource(w, req, requestInfo, crdInfo, terminating, supportedTypes)
	default:
		http.Error(w, "the server could not find the requested resource", http.StatusNotFound)
	}

	if handler != nil {
		handler = metrics.InstrumentHandlerFunc(verb, resource, subresource, scope, handler)
		handler(w, req)
		return
	}
}

func (r *crdHandler) serveResource(w http.ResponseWriter, req *http.Request, requestInfo *apirequest.RequestInfo, crdInfo *crdInfo, terminating bool, supportedTypes []string) http.HandlerFunc {
	requestScope := crdInfo.requestScopes[requestInfo.APIVersion]
	storage := crdInfo.storages[requestInfo.APIVersion].CustomResource
	minRequestTimeout := 1 * time.Minute

	switch requestInfo.Verb {
	case "get":
		return handlers.GetResource(storage, storage, requestScope)
	case "list":
		forceWatch := false
		return handlers.ListResource(storage, storage, requestScope, forceWatch, minRequestTimeout)
	case "watch":
		forceWatch := true
		return handlers.ListResource(storage, storage, requestScope, forceWatch, minRequestTimeout)
	case "create":
		if terminating {
			http.Error(w, fmt.Sprintf("%v not allowed while CustomResourceDefinition is terminating", requestInfo.Verb), http.StatusMethodNotAllowed)
			return nil
		}
		return handlers.CreateResource(storage, requestScope, r.admission)
	case "update":
		return handlers.UpdateResource(storage, requestScope, r.admission)
	case "patch":
		return handlers.PatchResource(storage, requestScope, r.admission, supportedTypes)
	case "delete":
		allowsOptions := true
		return handlers.DeleteResource(storage, allowsOptions, requestScope, r.admission)
	case "deletecollection":
		checkBody := true
		return handlers.DeleteCollection(storage, checkBody, requestScope, r.admission)
	default:
		http.Error(w, fmt.Sprintf("unhandled verb %q", requestInfo.Verb), http.StatusMethodNotAllowed)
		return nil
	}
}

func (r *crdHandler) serveStatus(w http.ResponseWriter, req *http.Request, requestInfo *apirequest.RequestInfo, crdInfo *crdInfo, terminating bool, supportedTypes []string) http.HandlerFunc {
	requestScope := crdInfo.statusRequestScopes[requestInfo.APIVersion]
	storage := crdInfo.storages[requestInfo.APIVersion].Status

	switch requestInfo.Verb {
	case "get":
		return handlers.GetResource(storage, nil, requestScope)
	case "update":
		return handlers.UpdateResource(storage, requestScope, r.admission)
	case "patch":
		return handlers.PatchResource(storage, requestScope, r.admission, supportedTypes)
	default:
		http.Error(w, fmt.Sprintf("unhandled verb %q", requestInfo.Verb), http.StatusMethodNotAllowed)
		return nil
	}
}

func (r *crdHandler) serveScale(w http.ResponseWriter, req *http.Request, requestInfo *apirequest.RequestInfo, crdInfo *crdInfo, terminating bool, supportedTypes []string) http.HandlerFunc {
	requestScope := crdInfo.scaleRequestScopes[requestInfo.APIVersion]
	storage := crdInfo.storages[requestInfo.APIVersion].Scale

	switch requestInfo.Verb {
	case "get":
		return handlers.GetResource(storage, nil, requestScope)
	case "update":
		return handlers.UpdateResource(storage, requestScope, r.admission)
	case "patch":
		return handlers.PatchResource(storage, requestScope, r.admission, supportedTypes)
	default:
		http.Error(w, fmt.Sprintf("unhandled verb %q", requestInfo.Verb), http.StatusMethodNotAllowed)
		return nil
	}
}

func (r *crdHandler) updateCustomResourceDefinition(oldObj, newObj interface{}) {
	oldCRD := oldObj.(*apiextensions.CustomResourceDefinition)
	newCRD := newObj.(*apiextensions.CustomResourceDefinition)

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap := r.customStorage.Load().(crdStorageMap)
	oldInfo, found := storageMap[newCRD.UID]
	if !found {
		return
	}
	if apiequality.Semantic.DeepEqual(&newCRD.Spec, oldInfo.spec) && apiequality.Semantic.DeepEqual(&newCRD.Status.AcceptedNames, oldInfo.acceptedNames) {
		glog.V(6).Infof("Ignoring customresourcedefinition %s update because neither spec, nor accepted names changed", oldCRD.Name)
		return
	}

	glog.V(4).Infof("Updating customresourcedefinition %s", oldCRD.Name)

	// Copy because we cannot write to storageMap without a race
	// as it is used without locking elsewhere.
	storageMap2 := storageMap.clone()
	if oldInfo, ok := storageMap2[types.UID(oldCRD.UID)]; ok {
		for _, storage := range oldInfo.storages {
			// destroy only the main storage. Those for the subresources share cacher and etcd clients.
			storage.CustomResource.DestroyFunc()
		}
		delete(storageMap2, types.UID(oldCRD.UID))
	}

	r.customStorage.Store(storageMap2)
}

// removeDeadStorage removes REST storage that isn't being used
func (r *crdHandler) removeDeadStorage() {
	allCustomResourceDefinitions, err := r.crdLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap := r.customStorage.Load().(crdStorageMap)
	// Copy because we cannot write to storageMap without a race
	// as it is used without locking elsewhere
	storageMap2 := storageMap.clone()
	for uid, s := range storageMap2 {
		found := false
		for _, crd := range allCustomResourceDefinitions {
			if crd.UID == uid {
				found = true
				break
			}
		}
		if !found {
			glog.V(4).Infof("Removing dead CRD storage for %s/%s", s.spec.Group, s.spec.Names.Kind)
			for _, storage := range s.storages {
				// destroy only the main storage. Those for the subresources share cacher and etcd clients.
				storage.CustomResource.DestroyFunc()
			}
			delete(storageMap2, uid)
		}
	}
	r.customStorage.Store(storageMap2)
}

// GetCustomResourceListerCollectionDeleter returns the ListerCollectionDeleter of
// the given crd.
func (r *crdHandler) GetCustomResourceListerCollectionDeleter(crd *apiextensions.CustomResourceDefinition) (finalizer.ListerCollectionDeleter, error) {
	info, err := r.getOrCreateServingInfoFor(crd)
	if err != nil {
		return nil, err
	}
	return info.storages[info.storageVersion].CustomResource, nil
}

func (r *crdHandler) getOrCreateServingInfoFor(crd *apiextensions.CustomResourceDefinition) (*crdInfo, error) {
	storageMap := r.customStorage.Load().(crdStorageMap)
	if ret, ok := storageMap[crd.UID]; ok {
		return ret, nil
	}

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap = r.customStorage.Load().(crdStorageMap)
	if ret, ok := storageMap[crd.UID]; ok {
		return ret, nil
	}

	storageVersion, err := apiextensions.GetCRDStorageVersion(crd)
	if err != nil {
		return nil, err
	}

	// Scope/Storages per version.
	requestScopes := map[string]handlers.RequestScope{}
	storages := map[string]customresource.CustomResourceStorage{}
	statusScopes := map[string]handlers.RequestScope{}
	scaleScopes := map[string]handlers.RequestScope{}

	for _, v := range crd.Spec.Versions {
		safeConverter, unsafeConverter := conversion.NewCRDConverter(crd)
		// In addition to Unstructured objects (Custom Resources), we also may sometimes need to
		// decode unversioned Options objects, so we delegate to parameterScheme for such types.
		parameterScheme := runtime.NewScheme()
		parameterScheme.AddUnversionedTypes(schema.GroupVersion{Group: crd.Spec.Group, Version: v.Name},
			&metav1.ListOptions{},
			&metav1.ExportOptions{},
			&metav1.GetOptions{},
			&metav1.DeleteOptions{},
		)
		parameterCodec := runtime.NewParameterCodec(parameterScheme)

		kind := schema.GroupVersionKind{Group: crd.Spec.Group, Version: v.Name, Kind: crd.Status.AcceptedNames.Kind}
		typer := newUnstructuredObjectTyper(parameterScheme)
		creator := unstructuredCreator{}

		validator, _, err := apiservervalidation.NewSchemaValidator(crd.Spec.Validation)
		if err != nil {
			return nil, err
		}

		var statusSpec *apiextensions.CustomResourceSubresourceStatus
		var statusValidator *validate.SchemaValidator
		if utilfeature.DefaultFeatureGate.Enabled(apiextensionsfeatures.CustomResourceSubresources) && crd.Spec.Subresources != nil && crd.Spec.Subresources.Status != nil {
			statusSpec = crd.Spec.Subresources.Status

			// for the status subresource, validate only against the status schema
			if crd.Spec.Validation != nil && crd.Spec.Validation.OpenAPIV3Schema != nil && crd.Spec.Validation.OpenAPIV3Schema.Properties != nil {
				if statusSchema, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["status"]; ok {
					openapiSchema := &spec.Schema{}
					if err := apiservervalidation.ConvertJSONSchemaProps(&statusSchema, openapiSchema); err != nil {
						return nil, err
					}
					statusValidator = validate.NewSchemaValidator(openapiSchema, nil, "", strfmt.Default)
				}
			}
		}

		var scaleSpec *apiextensions.CustomResourceSubresourceScale
		if utilfeature.DefaultFeatureGate.Enabled(apiextensionsfeatures.CustomResourceSubresources) && crd.Spec.Subresources != nil && crd.Spec.Subresources.Scale != nil {
			scaleSpec = crd.Spec.Subresources.Scale
		}

		// TODO: identify how to pass printer specification from the CRD
		table, err := tableconvertor.New(nil)
		if err != nil {
			glog.V(2).Infof("The CRD for %v has an invalid printer specification, falling back to default printing: %v", kind, err)
		}

		storages[v.Name] = customresource.NewStorage(
			schema.GroupResource{Group: crd.Spec.Group, Resource: crd.Status.AcceptedNames.Plural},
			schema.GroupVersionKind{Group: crd.Spec.Group, Version: v.Name, Kind: crd.Status.AcceptedNames.ListKind},
			customresource.NewStrategy(
				typer,
				crd.Spec.Scope == apiextensions.NamespaceScoped,
				kind,
				validator,
				statusValidator,
				statusSpec,
				scaleSpec,
			),
			crdConversionRESTOptionsGetter{
				RESTOptionsGetter: r.restOptionsGetter,
				converter:         safeConverter,
				decoderVersion:    schema.GroupVersion{Group: crd.Spec.Group, Version: v.Name},
				encoderVersion:    schema.GroupVersion{Group: crd.Spec.Group, Version: storageVersion},
			},
			crd.Status.AcceptedNames.Categories,
			table,
		)

		selfLinkPrefix := ""
		switch crd.Spec.Scope {
		case apiextensions.ClusterScoped:
			selfLinkPrefix = "/" + path.Join("apis", crd.Spec.Group, v.Name) + "/" + crd.Status.AcceptedNames.Plural + "/"
		case apiextensions.NamespaceScoped:
			selfLinkPrefix = "/" + path.Join("apis", crd.Spec.Group, v.Name, "namespaces") + "/"
		}

		clusterScoped := crd.Spec.Scope == apiextensions.ClusterScoped

		requestScopes[v.Name] = handlers.RequestScope{
			Namer: handlers.ContextBasedNaming{
				SelfLinker:         meta.NewAccessor(),
				ClusterScoped:      clusterScoped,
				SelfLinkPathPrefix: selfLinkPrefix,
			},
			Serializer:     unstructuredNegotiatedSerializer{typer: typer, creator: creator, converter: safeConverter},
			ParameterCodec: parameterCodec,

			Creater:         creator,
			Convertor:       safeConverter,
			Defaulter:       unstructuredDefaulter{parameterScheme},
			Typer:           typer,
			UnsafeConvertor: unsafeConverter,

			Resource: schema.GroupVersionResource{Group: crd.Spec.Group, Version: v.Name, Resource: crd.Status.AcceptedNames.Plural},
			Kind:     kind,

			MetaGroupVersion: metav1.SchemeGroupVersion,

			TableConvertor: storages[v.Name].CustomResource,
		}

		// override scaleSpec subresource values
		// shallow copy
		scaleScope := requestScopes[v.Name]
		scaleConverter := scale.NewScaleConverter()
		scaleScope.Subresource = "scale"
		scaleScope.Serializer = serializer.NewCodecFactory(scaleConverter.Scheme())
		scaleScope.Kind = autoscalingv1.SchemeGroupVersion.WithKind("Scale")
		scaleScope.Namer = handlers.ContextBasedNaming{
			SelfLinker:         meta.NewAccessor(),
			ClusterScoped:      clusterScoped,
			SelfLinkPathPrefix: selfLinkPrefix,
			SelfLinkPathSuffix: "/scale",
		}
		scaleScopes[v.Name] = scaleScope

		// override status subresource values
		// shallow copy
		statusScope := requestScopes[v.Name]
		statusScope.Subresource = "status"
		statusScope.Namer = handlers.ContextBasedNaming{
			SelfLinker:         meta.NewAccessor(),
			ClusterScoped:      clusterScoped,
			SelfLinkPathPrefix: selfLinkPrefix,
			SelfLinkPathSuffix: "/status",
		}
		statusScopes[v.Name] = statusScope
	}

	ret := &crdInfo{
		spec:                &crd.Spec,
		acceptedNames:       &crd.Status.AcceptedNames,
		storages:            storages,
		requestScopes:       requestScopes,
		scaleRequestScopes:  scaleScopes,
		statusRequestScopes: statusScopes,
		storageVersion:      storageVersion,
	}

	// Copy because we cannot write to storageMap without a race
	// as it is used without locking elsewhere.
	storageMap2 := storageMap.clone()

	storageMap2[crd.UID] = ret
	r.customStorage.Store(storageMap2)

	return ret, nil
}

type unstructuredNegotiatedSerializer struct {
	typer     runtime.ObjectTyper
	creator   runtime.ObjectCreater
	converter runtime.ObjectConvertor
}

func (s unstructuredNegotiatedSerializer) SupportedMediaTypes() []runtime.SerializerInfo {
	return []runtime.SerializerInfo{
		{
			MediaType:        "application/json",
			EncodesAsText:    true,
			Serializer:       json.NewSerializer(json.DefaultMetaFactory, s.creator, s.typer, false),
			PrettySerializer: json.NewSerializer(json.DefaultMetaFactory, s.creator, s.typer, true),
			StreamSerializer: &runtime.StreamSerializerInfo{
				EncodesAsText: true,
				Serializer:    json.NewSerializer(json.DefaultMetaFactory, s.creator, s.typer, false),
				Framer:        json.Framer,
			},
		},
		{
			MediaType:     "application/yaml",
			EncodesAsText: true,
			Serializer:    json.NewYAMLSerializer(json.DefaultMetaFactory, s.creator, s.typer),
		},
	}
}

func (s unstructuredNegotiatedSerializer) EncoderForVersion(encoder runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	return versioning.NewCodec(encoder, nil, s.converter, Scheme, Scheme, Scheme, gv, nil)
}

func (s unstructuredNegotiatedSerializer) DecoderToVersion(decoder runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder {
	return versioning.NewDefaultingCodecForScheme(Scheme, nil, decoder, nil, gv)
}

type UnstructuredObjectTyper struct {
	Delegate          runtime.ObjectTyper
	UnstructuredTyper runtime.ObjectTyper
}

func newUnstructuredObjectTyper(Delegate runtime.ObjectTyper) UnstructuredObjectTyper {
	return UnstructuredObjectTyper{
		Delegate:          Delegate,
		UnstructuredTyper: discovery.NewUnstructuredObjectTyper(),
	}
}

func (t UnstructuredObjectTyper) ObjectKinds(obj runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	// Delegate for things other than Unstructured.
	if _, ok := obj.(runtime.Unstructured); !ok {
		return t.Delegate.ObjectKinds(obj)
	}
	return t.UnstructuredTyper.ObjectKinds(obj)
}

func (t UnstructuredObjectTyper) Recognizes(gvk schema.GroupVersionKind) bool {
	return t.Delegate.Recognizes(gvk) || t.UnstructuredTyper.Recognizes(gvk)
}

type unstructuredCreator struct{}

func (c unstructuredCreator) New(kind schema.GroupVersionKind) (runtime.Object, error) {
	ret := &unstructured.Unstructured{}
	ret.SetGroupVersionKind(kind)
	return ret, nil
}

type unstructuredDefaulter struct {
	delegate runtime.ObjectDefaulter
}

func (d unstructuredDefaulter) Default(in runtime.Object) {
	// Delegate for things other than Unstructured.
	if _, ok := in.(runtime.Unstructured); !ok {
		d.delegate.Default(in)
	}
}

type CRDRESTOptionsGetter struct {
	StorageConfig           storagebackend.Config
	StoragePrefix           string
	EnableWatchCache        bool
	DefaultWatchCacheSize   int
	EnableGarbageCollection bool
	DeleteCollectionWorkers int
}

func (t CRDRESTOptionsGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	ret := generic.RESTOptions{
		StorageConfig:           &t.StorageConfig,
		Decorator:               generic.UndecoratedStorage,
		EnableGarbageCollection: t.EnableGarbageCollection,
		DeleteCollectionWorkers: t.DeleteCollectionWorkers,
		ResourcePrefix:          resource.Group + "/" + resource.Resource,
	}
	if t.EnableWatchCache {
		ret.Decorator = genericregistry.StorageWithCacher(t.DefaultWatchCacheSize)
	}
	return ret, nil
}

// clone returns a clone of the provided crdStorageMap.
// The clone is a shallow copy of the map.
func (in crdStorageMap) clone() crdStorageMap {
	if in == nil {
		return nil
	}
	out := make(crdStorageMap, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// crdConversionRESTOptionsGetter overrides the codec with one using the
// provided custom converter and custom encoder and decoder version.
type crdConversionRESTOptionsGetter struct {
	generic.RESTOptionsGetter
	converter      runtime.ObjectConvertor
	encoderVersion schema.GroupVersion
	decoderVersion schema.GroupVersion
}

func (t crdConversionRESTOptionsGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	ret, err := t.RESTOptionsGetter.GetRESTOptions(resource)
	if err == nil {
		ret.StorageConfig.Codec = versioning.NewCodec(ret.StorageConfig.Codec, ret.StorageConfig.Codec, t.converter, &unstructuredCreator{}, discovery.NewUnstructuredObjectTyper(), &unstructuredDefaulter{delegate: Scheme}, t.encoderVersion, t.decoderVersion)
	}
	return ret, err
}
