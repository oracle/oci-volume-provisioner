// Copyright (c) 2016, 2017, Oracle and/or its affiliates. All rights reserved.
// Code generated. DO NOT EDIT.

// File Storage Service API
//
// APIs for OCI file storage service.
//

package ffsw

import(
    "github.com/oracle/oci-go-sdk/common"
    "context"
    "fmt"
    "net/http"
)

//FileStorageClient a client for FileStorage
type FileStorageClient struct {
    common.BaseClient
    config *common.ConfigurationProvider
}


// NewFileStorageClientWithConfigurationProvider Creates a new default FileStorage client with the given configuration provider.
// the configuration provider will be used for the default signer as well as reading the region
func NewFileStorageClientWithConfigurationProvider(configProvider common.ConfigurationProvider) (client FileStorageClient, err error){
    baseClient, err := common.NewClientWithConfig(configProvider)
    if err != nil {
        return
    }

    client = FileStorageClient{BaseClient: baseClient}
    client.BasePath = "20171215"
    err = client.setConfigurationProvider(configProvider)
    return
}


// SetConfigurationProvider sets the configuration provider, returns an error if is not valid
func (client *FileStorageClient) setConfigurationProvider(configProvider common.ConfigurationProvider) error {
    if ok, err := common.IsConfigurationProviderValid(configProvider); !ok {
        return err
    }

    region, err := configProvider.Region()
    if err != nil {
        return err
    }
    client.config = &configProvider
    client.Host = fmt.Sprintf(common.DefaultHostURLTemplate, "filestorage", string(region))
    return nil
}

// ConfigurationProvider the ConfigurationProvider used in this client, or null if none set
func (client *FileStorageClient) ConfigurationProvider() *common.ConfigurationProvider {
    return client.config
}





 // CreateExport Creates a new export in the specifed export set, path, and
 // file system.
func(client FileStorageClient) CreateExport(ctx context.Context, request CreateExportRequest, options ...common.RetryPolicyOption) (response CreateExportResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodPost, "/exports", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // CreateFileSystem Create a new file system in the specified compartment and
 // availability domain. File systems in one availability domain
 // can be mounted by instances in another availability domain,
 // but they may see higher latencies than instances in the same
 // availability domain as the file system.
 // Once a file system is created it can be assocated with a mount
 // target and then mounted by instances that can connect to the
 // mount target's IP address. A file system can be assocated with
 // more than one mount target at a time.
 // For information about access control and compartments, see
 // [Overview of the IAM Service]({{DOC_SERVER_URL}}/Content/Identity/Concepts/overview.htm).
 // For information about Availability Domains, see [Regions and
 // Availability Domains]({{DOC_SERVER_URL}}/Content/General/Concepts/regions.htm).
 // To get a list of Availability Domains, use the
 // `ListAvailabilityDomains` operation in the Identity and Access
 // Management Service API.
 // All Oracle Bare Metal Cloud Services resources, including
 // file systems, get an Oracle-assigned, unique ID called an Oracle
 // Cloud Identifier (OCID).  When you create a resource, you can
 // find its OCID in the response. You can also retrieve a
 // resource's OCID by using a List API operation on that resource
 // type, or by viewing the resource in the Console.
func(client FileStorageClient) CreateFileSystem(ctx context.Context, request CreateFileSystemRequest, options ...common.RetryPolicyOption) (response CreateFileSystemResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodPost, "/fileSystems", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // CreateMountTarget Create a new mount target in the specified compartment and
 // subnet. A file system can only be assocated with a mount
 // target if they are both in the availablity domain. Instances
 // can connect to mount targets in another availablity domain but
 // they may see higher latencies than instances in the same
 // availability domain as the mount target.
 // Mount targets have one or more private IP addresses that can
 // be used as the host portion of remotetarget parameters in
 // client mount commands. These private IP addresses are listed
 // in privateIpIds property of the mount target and are HA. Mount
 // targets also consume additional IP addresses in their subnet.
 // For information about access control and compartments, see
 // [Overview of the IAM
 // Service]({{DOC_SERVER_URL}}/Content/Identity/Concepts/overview.htm).
 // For information about Availability Domains, see [Regions and
 // Availability Domains]({{DOC_SERVER_URL}}/Content/General/Concepts/regions.htm).
 // To get a list of Availability Domains, use the
 // `ListAvailabilityDomains` operation in the Identity and Access
 // Management Service API.
 // All Oracle Bare Metal Cloud Services resources, including
 // mount targets, get an Oracle-assigned, unique ID called an
 // Oracle Cloud Identifier (OCID).  When you create a resource,
 // you can find its OCID in the response. You can also retrieve a
 // resource's OCID by using a List API operation on that resource
 // type, or by viewing the resource in the Console.
func(client FileStorageClient) CreateMountTarget(ctx context.Context, request CreateMountTargetRequest, options ...common.RetryPolicyOption) (response CreateMountTargetResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodPost, "/mountTargets", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // CreateSnapshot Creates a new snapshot of the specified file system. The
 // snapshot will be accessible at `.snapshot/<name>`.
func(client FileStorageClient) CreateSnapshot(ctx context.Context, request CreateSnapshotRequest, options ...common.RetryPolicyOption) (response CreateSnapshotResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodPost, "/snapshots", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // DeleteExport Delete the specified export.
func(client FileStorageClient) DeleteExport(ctx context.Context, request DeleteExportRequest, options ...common.RetryPolicyOption) ( err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodDelete, "/exports/{exportId}", request)
        if err != nil {
            return
        }


    _, err = client.Call(ctx, &httpRequest)
    return
}



 // DeleteFileSystem Delete the specified file system. The file system must not be
 // referenced by any non-deleted export resources. Deleting a
 // file system also deletes all of its snapshots.
func(client FileStorageClient) DeleteFileSystem(ctx context.Context, request DeleteFileSystemRequest, options ...common.RetryPolicyOption) ( err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodDelete, "/fileSystems/{fileSystemId}", request)
        if err != nil {
            return
        }


    _, err = client.Call(ctx, &httpRequest)
    return
}



 // DeleteMountTarget Delete the specified mount target. This will also delete the
 // mount target's VNICs.
func(client FileStorageClient) DeleteMountTarget(ctx context.Context, request DeleteMountTargetRequest, options ...common.RetryPolicyOption) ( err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodDelete, "/mountTargets/{mountTargetId}", request)
        if err != nil {
            return
        }


    _, err = client.Call(ctx, &httpRequest)
    return
}



 // DeleteSnapshot Delete the specified snapshot.
func(client FileStorageClient) DeleteSnapshot(ctx context.Context, request DeleteSnapshotRequest, options ...common.RetryPolicyOption) ( err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodDelete, "/snapshots/{snapshotId}", request)
        if err != nil {
            return
        }


    _, err = client.Call(ctx, &httpRequest)
    return
}



 // GetExport Gets the specified export's information.
func(client FileStorageClient) GetExport(ctx context.Context, request GetExportRequest, options ...common.RetryPolicyOption) (response GetExportResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/exports/{exportId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // GetExportSet Gets the specified export set's information.
func(client FileStorageClient) GetExportSet(ctx context.Context, request GetExportSetRequest, options ...common.RetryPolicyOption) (response GetExportSetResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/exportSets/{exportSetId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // GetFileSystem Gets the specified file system's information.
func(client FileStorageClient) GetFileSystem(ctx context.Context, request GetFileSystemRequest, options ...common.RetryPolicyOption) (response GetFileSystemResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/fileSystems/{fileSystemId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // GetMountTarget Gets the specified mount targets's information.
func(client FileStorageClient) GetMountTarget(ctx context.Context, request GetMountTargetRequest, options ...common.RetryPolicyOption) (response GetMountTargetResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/mountTargets/{mountTargetId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // GetSnapshot Gets the specified snapshot's information.
func(client FileStorageClient) GetSnapshot(ctx context.Context, request GetSnapshotRequest, options ...common.RetryPolicyOption) (response GetSnapshotResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/snapshots/{snapshotId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // ListExportSets List the export set resources in the specified compartment.
func(client FileStorageClient) ListExportSets(ctx context.Context, request ListExportSetsRequest, options ...common.RetryPolicyOption) (response ListExportSetsResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/exportSets", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // ListExports List the export resources in the specified compartment. Must
 // also specify an export set and / or a file system.
func(client FileStorageClient) ListExports(ctx context.Context, request ListExportsRequest, options ...common.RetryPolicyOption) (response ListExportsResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/exports", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // ListFileSystems List the file system resources in the specified compartment.
func(client FileStorageClient) ListFileSystems(ctx context.Context, request ListFileSystemsRequest, options ...common.RetryPolicyOption) (response ListFileSystemsResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/fileSystems", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // ListLockOwners List the lock owners in a given file system.
func(client FileStorageClient) ListLockOwners(ctx context.Context, request ListLockOwnersRequest, options ...common.RetryPolicyOption) (response ListLockOwnersResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/fileSystems/{fileSystemId}/lockOwners", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // ListMountTargets List the mount target resources in the specified compartment.
func(client FileStorageClient) ListMountTargets(ctx context.Context, request ListMountTargetsRequest, options ...common.RetryPolicyOption) (response ListMountTargetsResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/mountTargets", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // ListSnapshots List the snapshots of the specified file system.
func(client FileStorageClient) ListSnapshots(ctx context.Context, request ListSnapshotsRequest, options ...common.RetryPolicyOption) (response ListSnapshotsResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodGet, "/snapshots", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // UpdateExportSet Update the specified export set's information.
func(client FileStorageClient) UpdateExportSet(ctx context.Context, request UpdateExportSetRequest, options ...common.RetryPolicyOption) (response UpdateExportSetResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodPut, "/exportSets/{exportSetId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // UpdateFileSystem Update the specified file system's information.
func(client FileStorageClient) UpdateFileSystem(ctx context.Context, request UpdateFileSystemRequest, options ...common.RetryPolicyOption) (response UpdateFileSystemResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodPut, "/fileSystems/{fileSystemId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}



 // UpdateMountTarget Update the specified mount targets's information.
func(client FileStorageClient) UpdateMountTarget(ctx context.Context, request UpdateMountTargetRequest, options ...common.RetryPolicyOption) (response UpdateMountTargetResponse,  err error) {
        httpRequest, err := common.MakeDefaultHTTPRequestWithTaggedStruct(http.MethodPut, "/mountTargets/{mountTargetId}", request)
        if err != nil {
            return
        }


    httpResponse, err := client.Call(ctx, &httpRequest, options...)
    defer common.CloseBodyIfValid(httpResponse)
    response.RawResponse = httpResponse
    if err != nil {
        return
    }

        err = common.UnmarshalResponse(httpResponse, &response)
    return
}

