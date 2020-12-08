// +build test

package task

import (
	"math/rand"
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
	interleavedWeightedRoundRobinSchedulerSuite struct {
		*require.Assertions
		suite.Suite

		controller    *gomock.Controller
		mockProcessor *MockProcessor

		scheduler *InterleavedWeightedRoundRobinScheduler
	}
)

func TestInterleavedWeightedRoundRobinSchedulerSuite(t *testing.T) {
	s := new(interleavedWeightedRoundRobinSchedulerSuite)
	suite.Run(t, s)
}

func (s *interleavedWeightedRoundRobinSchedulerSuite) SetupSuite() {

}

func (s *interleavedWeightedRoundRobinSchedulerSuite) TearDownSuite() {

}

func (s *interleavedWeightedRoundRobinSchedulerSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockProcessor = NewMockProcessor(s.controller)

	priorityToWeight := map[int]int{
		0: 5,
		1: 3,
		2: 2,
		3: 1,
	}
	logger := loggerimpl.NewDevelopmentForTest(s.Suite)
	metricsClient := metrics.NewClient(tally.NoopScope, metrics.Common)

	s.scheduler = NewInterleavedWeightedRoundRobin(
		2,
		priorityToWeight,
		s.mockProcessor,
		logger,
		metricsClient,
	)
}

func (s *interleavedWeightedRoundRobinSchedulerSuite) TearDownTest() {
	s.scheduler.Stop()
	s.controller.Finish()
}

func (s *interleavedWeightedRoundRobinSchedulerSuite) TestSubmitSchedule_Success() {
	s.mockProcessor.EXPECT().Start()
	s.scheduler.Start()
	s.mockProcessor.EXPECT().Stop()

	testWaitGroup := sync.WaitGroup{}
	testWaitGroup.Add(1)

	mockTask := NewMockPriorityTask(s.controller)
	mockTask.EXPECT().Priority().Return(0).AnyTimes()
	s.mockProcessor.EXPECT().Submit(mockTask).Do(func(task Task) {
		testWaitGroup.Done()
	})

	s.scheduler.Submit(mockTask)

	testWaitGroup.Wait()
}

func (s *interleavedWeightedRoundRobinSchedulerSuite) TestSubmitSchedule_Fail() {
	s.mockProcessor.EXPECT().Start()
	s.scheduler.Start()
	s.mockProcessor.EXPECT().Stop()
	s.scheduler.Stop()

	testWaitGroup := sync.WaitGroup{}
	testWaitGroup.Add(1)

	mockTask := NewMockPriorityTask(s.controller)
	mockTask.EXPECT().Priority().Return(0).AnyTimes()
	// either drain immediately
	mockTask.EXPECT().Reschedule().Do(func() {
		testWaitGroup.Done()
	}).MaxTimes(1)
	// or process by worker
	s.mockProcessor.EXPECT().Submit(mockTask).Do(func(task Task) {
		testWaitGroup.Done()
	}).MaxTimes(1)

	s.scheduler.Submit(mockTask)

	testWaitGroup.Wait()
}

func (s *interleavedWeightedRoundRobinSchedulerSuite) TestSequentialSubmitSchedule_Case1() {
	testWaitGroup := sync.WaitGroup{}

	// 2 task from priority 0, 5 weights
	// 3 task from priority 1, 3 weights
	// 1 task from priority 2, 2 weights
	// 1 task from priority 3, 1 weights

	mockTasks := []*PriorityTask{}

	for i := 0; i < 2; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(0).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	for i := 0; i < 3; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(1).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	for i := 0; i < 1; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(2).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	for i := 0; i < 1; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(3).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	expectedPriorities := make(chan int, 7)
	expectedPriorities <- 0
	expectedPriorities <- 0
	expectedPriorities <- 1
	expectedPriorities <- 1
	expectedPriorities <- 2
	expectedPriorities <- 1
	expectedPriorities <- 3

	for _, mockTask := range mockTasks {
		s.mockProcessor.EXPECT().Submit(mockTask).Do(func(task Task) {
			testWaitGroup.Done()
			s.Equal(task.(PriorityTask).Priority(), <-expectedPriorities)
		}).Times(1)
	}

	s.mockProcessor.EXPECT().Start()
	s.scheduler.Start()
	s.mockProcessor.EXPECT().Stop()

	testWaitGroup.Wait()
}

func (s *interleavedWeightedRoundRobinSchedulerSuite) TestSequentialSubmitSchedule_Case2() {
	testWaitGroup := sync.WaitGroup{}

	// 8 task from priority 0, 5 weights
	// 4 task from priority 1, 3 weights
	// 5 task from priority 2, 2 weights
	// 2 task from priority 3, 1 weights

	mockTasks := []*PriorityTask{}

	for i := 0; i < 8; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(0).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	for i := 0; i < 4; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(1).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	for i := 0; i < 5; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(2).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	for i := 0; i < 2; i++ {
		testWaitGroup.Add(1)
		mockTask := NewMockPriorityTask(s.controller)
		mockTask.EXPECT().Priority().Return(3).AnyTimes()
		mockTasks = append(mockTasks, mockTask)
	}

	expectedPriorities := make(chan int, 19)
	expectedPriorities <- 0
	expectedPriorities <- 0
	expectedPriorities <- 0
	expectedPriorities <- 1
	expectedPriorities <- 0
	expectedPriorities <- 1
	expectedPriorities <- 2
	expectedPriorities <- 0
	expectedPriorities <- 1
	expectedPriorities <- 2
	expectedPriorities <- 3
	expectedPriorities <- 0
	expectedPriorities <- 0
	expectedPriorities <- 0
	expectedPriorities <- 1
	expectedPriorities <- 2
	expectedPriorities <- 2
	expectedPriorities <- 3

	for _, mockTask := range mockTasks {
		s.mockProcessor.EXPECT().Submit(mockTask).Do(func(task Task) {
			testWaitGroup.Done()
			s.Equal(task.(PriorityTask).Priority(), <-expectedPriorities)
		}).Times(1)
	}

	s.mockProcessor.EXPECT().Start()
	s.scheduler.Start()
	s.mockProcessor.EXPECT().Stop()

	testWaitGroup.Wait()
}

func (s *interleavedWeightedRoundRobinSchedulerSuite) TestParallelSubmitSchedule() {
	s.mockProcessor.EXPECT().Start()
	s.scheduler.Start()
	s.mockProcessor.EXPECT().Stop()

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
			mockTask := NewMockPriorityTask(s.controller)
			priority := rand.Intn(4)
			mockTask.EXPECT().Priority().Return(priority).AnyTimes()
			s.mockProcessor.EXPECT().Submit(mockTask).Do(func(task Task) {
				testWaitGroup.Done()
			}).Times(1)
			channel <- mockTask
		}
		close(channel)

		go func() {
			startWaitGroud.Wait()

			for mockTask := range channel {
				s.scheduler.Submit(mockTask)
			}

			endWaitGroud.Done()
		}()

		startWaitGroud.Done()
	}
	endWaitGroud.Wait()

	testWaitGroup.Wait()
}
