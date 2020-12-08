// +build test

package task

import (
	"errors"
	"sync"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber-go/tally"
	"go.temporal.io/server/common/log/loggerimpl"
	"go.temporal.io/server/common/metrics"
)

type (
	parallelTaskProcessorSuite struct {
		*require.Assertions
		suite.Suite

		controller    *gomock.Controller
		mockProcessor *MockProcessor

		processor *ParallelTaskProcessor
	}
)

func TestParallelTaskProcessorSuite(t *testing.T) {
	s := new(parallelTaskProcessorSuite)
	suite.Run(t, s)
}

func (s *parallelTaskProcessorSuite) SetupSuite() {

}

func (s *parallelTaskProcessorSuite) TearDownSuite() {

}

func (s *parallelTaskProcessorSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockProcessor = NewMockProcessor(s.controller)

	logger := loggerimpl.NewDevelopmentForTest(s.Suite)
	metricsClient := metrics.NewClient(tally.NoopScope, metrics.Common)

	s.processor = NewParallelTaskProcessor(
		logger,
		metricsClient,
		nil,
	)
}

func (s *parallelTaskProcessorSuite) TearDownTest() {
	s.processor.Stop()
	s.controller.Finish()
}

func (s *parallelTaskProcessorSuite) TestSubmitProcess_Success() {
	testWaitGroup := sync.WaitGroup{}
	testWaitGroup.Add(1)

	mockTask := NewMockTask(s.controller)
	mockTask.EXPECT().RetryPolicy().Return(nil).AnyTimes()
	mockTask.EXPECT().Execute().Return(nil).Times(1)
	mockTask.EXPECT().Ack().Do(func() { testWaitGroup.Done() }).Times(1)

	s.processor.Submit(mockTask)

	testWaitGroup.Wait()
}

func (s *parallelTaskProcessorSuite) TestSubmitProcess_Fail() {
	testWaitGroup := sync.WaitGroup{}
	testWaitGroup.Add(1)

	s.processor.Stop()

	mockTask := NewMockTask(s.controller)
	// either drain immediately
	mockTask.EXPECT().Reschedule().Do(func() { testWaitGroup.Done() }).MaxTimes(1)
	// or process by worker
	mockTask.EXPECT().RetryPolicy().Return(nil).AnyTimes()
	mockTask.EXPECT().Execute().Return(nil).Times(1)
	mockTask.EXPECT().Ack().Do(func() { testWaitGroup.Done() }).MaxTimes(1)

	s.processor.Submit(mockTask)

	testWaitGroup.Wait()
}

func (s *parallelTaskProcessorSuite) TestSubmitProcess_FailExecutionAndStop() {
	testWaitGroup := sync.WaitGroup{}
	testWaitGroup.Add(1)

	s.processor.Stop()

	mockTask := NewMockTask(s.controller)
	mockTask.EXPECT().RetryPolicy().Return(nil).AnyTimes()
	executionErr := errors.New("random transient error")
	mockTask.EXPECT().Execute().Return(executionErr).Times(1)
	mockTask.EXPECT().HandleErr(executionErr).DoAndReturn(func(err error) error {
		s.processor.Stop()
		return err
	}).Times(1)
	mockTask.EXPECT().RetryErr(executionErr).Return(true).Times(1)
	mockTask.EXPECT().Reschedule().Do(func() { testWaitGroup.Done() }).Times(1)

	s.processor.Submit(mockTask)

	testWaitGroup.Wait()
}

func (s *parallelTaskProcessorSuite) TestParallelSubmitProcess() {
	numSubmitter := 200
	numTasks := 10

	testWaitGroup := sync.WaitGroup{}
	testWaitGroup.Add(numSubmitter * numTasks)

	startWaitGroud := sync.WaitGroup{}
	endWaitGroud := sync.WaitGroup{}

	startWaitGroud.Add(numSubmitter)

	for i := 0; i < numSubmitter; i++ {
		endWaitGroud.Add(1)

		channel := make(chan PriorityTask, 200)
		for i := 0; i < numTasks; i++ {
			mockTask := NewMockTask(s.controller)
			mockTask.EXPECT().RetryPolicy().Return(nil).AnyTimes()
			mockTask.EXPECT().Execute().Return(nil).Times(1)
			mockTask.EXPECT().Ack().Do(func() { testWaitGroup.Done() }).Times(1)
			channel <- mockTask
		}
		close(channel)

		go func() {
			startWaitGroud.Wait()

			for mockTask := range channel {
				s.processor.Submit(mockTask)
			}

			endWaitGroud.Done()
		}()

		startWaitGroud.Done()
	}
	endWaitGroud.Wait()

	testWaitGroup.Wait()
}
