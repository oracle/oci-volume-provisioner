// Copyright (c) 2016, 2017, Oracle and/or its affiliates. All rights reserved.
// Code generated. DO NOT EDIT.

// File Storage Service API
//
// APIs for OCI file storage service.
//

package ffsw

import (
    "context"
    "fmt"
    "time"
    oci_common "github.com/oracle/oci-go-sdk/common"
)


// PollExportUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollExportUntil(ctx context.Context, request GetExportRequest, predicate func(GetExportResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.GetExport(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollExportSetUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollExportSetUntil(ctx context.Context, request GetExportSetRequest, predicate func(GetExportSetResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.GetExportSet(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollExportSetSummaryUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollExportSetSummaryUntil(ctx context.Context, request ListExportSetsRequest, predicate func(ListExportSetsResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.ListExportSets(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollExportSummaryUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollExportSummaryUntil(ctx context.Context, request ListExportsRequest, predicate func(ListExportsResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.ListExports(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollFileSystemUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollFileSystemUntil(ctx context.Context, request GetFileSystemRequest, predicate func(GetFileSystemResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.GetFileSystem(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollFileSystemSummaryUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollFileSystemSummaryUntil(ctx context.Context, request ListFileSystemsRequest, predicate func(ListFileSystemsResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.ListFileSystems(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollMountTargetUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollMountTargetUntil(ctx context.Context, request GetMountTargetRequest, predicate func(GetMountTargetResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.GetMountTarget(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollMountTargetSummaryUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollMountTargetSummaryUntil(ctx context.Context, request ListMountTargetsRequest, predicate func(ListMountTargetsResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.ListMountTargets(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollSnapshotUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollSnapshotUntil(ctx context.Context, request GetSnapshotRequest, predicate func(GetSnapshotResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.GetSnapshot(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

// PollSnapshotSummaryUntil polls a resource until the specified predicate returns true
func (client FileStorageClient) PollSnapshotSummaryUntil(ctx context.Context, request ListSnapshotsRequest, predicate func(ListSnapshotsResponse, error) bool, options ...oci_common.RetryPolicyOption) error {
    policy := oci_common.BuildRetryPolicy(options...)
    deadlineContext, deadlineCancel := context.WithTimeout(ctx, oci_common.GetMaximumTimeout(policy))
    defer deadlineCancel()

    for currentOperationAttempt := uint(1); oci_common.ShouldContinueIssuingRequests(currentOperationAttempt, policy.MaximumNumberAttempts); currentOperationAttempt++ {
        response, err := client.ListSnapshots(deadlineContext, request)

        select {
        case <-deadlineContext.Done():
            return ctx.Err()
        default:
            // non-blocking select
        }

        if predicate(response, err) {
            return nil
        }

        if policy.ShouldRetryOperation(response.RawResponse, err, currentOperationAttempt) {
            if currentOperationAttempt != policy.MaximumNumberAttempts {
                // sleep before retrying the operation
                duration := policy.GetNextDuration(currentOperationAttempt)
                if deadline, ok := ctx.Deadline(); ok && time.Now().Add(duration).After(deadline) {
                    return oci_common.DurationExceedsDeadline
                }
                time.Sleep(duration)
            }
        } else {
            // we should NOT retry operation based on response and/or error => return
            return err
        }
    }
    return fmt.Errorf("maximum number of attempts exceeded (%v)", policy.MaximumNumberAttempts)
}

