// Copyright (c) 2016, 2017, Oracle and/or its affiliates. All rights reserved.
// Code generated. DO NOT EDIT.

package objectstorage

import (
    "github.com/oracle/oci-go-sdk/common"
    "net/http"
)

// AbortMultipartUploadRequest wrapper for the AbortMultipartUpload operation
type AbortMultipartUploadRequest struct {
        
 // The top-level namespace used for the request. 
        NamespaceName *string `mandatory:"true" contributesTo:"path" name:"namespaceName"`
        
 // The name of the bucket.
 // Example: `my-new-bucket1` 
        BucketName *string `mandatory:"true" contributesTo:"path" name:"bucketName"`
        
 // The name of the object.
 // Example: `test/object1.log` 
        ObjectName *string `mandatory:"true" contributesTo:"path" name:"objectName"`
        
 // The upload ID for a multipart upload. 
        UploadId *string `mandatory:"true" contributesTo:"query" name:"uploadId"`
        
 // The client request ID for tracing. 
        OpcClientRequestId *string `mandatory:"false" contributesTo:"header" name:"opc-client-request-id"`
}

func (request AbortMultipartUploadRequest) String() string {
    return common.PointerString(request)
}

// AbortMultipartUploadResponse wrapper for the AbortMultipartUpload operation
type AbortMultipartUploadResponse struct {

    // The underlying http response
    RawResponse *http.Response

    
 // Echoes back the value passed in the opc-client-request-id header, for use by clients when debugging.
    OpcClientRequestId *string `presentIn:"header" name:"opc-client-request-id"`
    
 // Unique Oracle-assigned identifier for the request. If you need to contact Oracle about a particular
 // request, please provide this request ID.
    OpcRequestId *string `presentIn:"header" name:"opc-request-id"`


}

func (response AbortMultipartUploadResponse) String() string {
    return common.PointerString(response)
}

