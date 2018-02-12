// Copyright (c) 2016, 2017, Oracle and/or its affiliates. All rights reserved.
// Code generated. DO NOT EDIT.

package database

import (
    "github.com/oracle/oci-go-sdk/common"
    "net/http"
)

// CreateDataGuardAssociationRequest wrapper for the CreateDataGuardAssociation operation
type CreateDataGuardAssociationRequest struct {
        
 // The database [OCID]({{DOC_SERVER_URL}}/Content/General/Concepts/identifiers.htm). 
        DatabaseId *string `mandatory:"true" contributesTo:"path" name:"databaseId"`
        
 // A request to create a Data Guard association. 
         CreateDataGuardAssociationDetails `contributesTo:"body"`
        
 // A token that uniquely identifies a request so it can be retried in case of a timeout or
 // server error without risk of executing that same action again. Retry tokens expire after 24
 // hours, but can be invalidated before then due to conflicting operations (for example, if a resource
 // has been deleted and purged from the system, then a retry of the original creation request
 // may be rejected). 
        OpcRetryToken *string `mandatory:"false" contributesTo:"header" name:"opc-retry-token"`
}

func (request CreateDataGuardAssociationRequest) String() string {
    return common.PointerString(request)
}

// CreateDataGuardAssociationResponse wrapper for the CreateDataGuardAssociation operation
type CreateDataGuardAssociationResponse struct {

    // The underlying http response
    RawResponse *http.Response
    
 // The DataGuardAssociation instance
     DataGuardAssociation `presentIn:"body"`

    
 // For optimistic concurrency control. See `if-match`.
    Etag *string `presentIn:"header" name:"etag"`
    
 // Unique Oracle-assigned identifier for the request. If you need to contact Oracle about
 // a particular request, please provide the request ID.
    OpcRequestId *string `presentIn:"header" name:"opc-request-id"`


}

func (response CreateDataGuardAssociationResponse) String() string {
    return common.PointerString(response)
}

