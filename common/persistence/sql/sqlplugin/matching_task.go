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

package sqlplugin

import (
	"context"
	"database/sql"
)

type (
	// TasksRow represents a row in tasks table
	TasksRow struct {
		RangeHash    uint32
		TaskQueueID  []byte
		TaskID       int64
		Data         []byte
		DataEncoding string
	}

	// TasksFilter contains the column names within tasks table that
	// can be used to filter results through a WHERE clause
	TasksFilter struct {
		RangeHash   uint32
		TaskQueueID []byte
		TaskID      int64
	}

	TasksRangeFilter struct {
		RangeHash   uint32
		TaskQueueID []byte
		MinTaskID   int64
		MaxTaskID   int64
		PageSize    int
	}

	// MatchingTask is the SQL persistence interface for matching tasks
	MatchingTask interface {
		// InsertIntoTasks insert one or more rows into tasks table.
		InsertIntoTasks(ctx context.Context, rows []TasksRow) (sql.Result, error)
		// SelectFromTasks returns rows from the tasks table.
		SelectFromTasks(ctx context.Context, filter TasksFilter) ([]TasksRow, error)
		// RangeSelectFromTasks returns rows from the tasks table.
		RangeSelectFromTasks(ctx context.Context, filter TasksRangeFilter) ([]TasksRow, error)
		// DeleteFromTasks deletes a row from tasks table.
		DeleteFromTasks(ctx context.Context, filter TasksFilter) (sql.Result, error)
		// RangeDeleteFromTasks deletes one or more rows from tasks table.
		RangeDeleteFromTasks(ctx context.Context, filter TasksRangeFilter) (sql.Result, error)
	}
)
