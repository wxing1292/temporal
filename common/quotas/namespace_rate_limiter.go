// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package quotas

import (
	"context"
	"time"
)

type (
	// NamespaceRateLimiterFn returns generate a namespace specific rate limiter
	NamespaceRateLimiterFn func(namespaceID string) RateLimiter

	// NamespaceRateLimiter corresponds to basic rate limiting functionality.
	NamespaceRateLimiter interface {
		// Allow attempts to allow a request to go through. The method returns
		// immediately with a true or false indicating if the request can make
		// progress
		Allow(namespaceID string) bool

		// AllowN attempts to allow a requests to go through. The method returns
		// immediately with a true or false indicating if the request can make
		// progress
		AllowN(namespaceID string, now time.Time, numToken int) bool

		// Reserve returns a Reservation that indicates how long the caller
		// must wait before event happen.
		Reserve(namespaceID string) Reservation

		// ReserveN returns a Reservation that indicates how long the caller
		// must wait before event happen.
		ReserveN(namespaceID string, now time.Time, numToken int) Reservation

		// Wait waits till the deadline for a rate limit token to allow the request
		// to go through.
		Wait(ctx context.Context, namespaceID string) error

		// WaitN waits till the deadline for a rate limit token to allow the request
		// to go through.
		WaitN(ctx context.Context, namespaceID string, numToken int) error
	}
)
