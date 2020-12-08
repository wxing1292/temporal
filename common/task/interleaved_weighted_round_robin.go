package task

import (
	"sort"
	"sync"
	"sync/atomic"

	"go.temporal.io/server/common"
	"go.temporal.io/server/common/backoff"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
)

type (
	// InterleavedWeightedRoundRobinSchedulerOptions configs
	// interleaved weighted round robin scheduler
	InterleavedWeightedRoundRobinSchedulerOptions struct {
		QueueSize   int
		WorkerCount int
		RetryPolicy backoff.RetryPolicy
	}

	InterleavedWeightedRoundRobinSchedulerImpl struct {
		status       int32
		processor    Processor
		logger       log.Logger
		metricsScope metrics.Scope

		notifyChan   chan struct{}
		shutdownChan chan struct{}

		priorityToWeight map[int]int

		sync.RWMutex
		weightToTaskChans map[int]*WeightedChan
	}
)

func NewInterleavedWeightedRoundRobin(
	chanSize int,
	priorityToWeight map[int]int,
	processor Processor,
	logger log.Logger,
	metricsClient metrics.Client,
) *InterleavedWeightedRoundRobinSchedulerImpl {
	return &InterleavedWeightedRoundRobinSchedulerImpl{
		status:       common.DaemonStatusInitialized,
		processor:    processor,
		logger:       logger,
		metricsScope: metricsClient.Scope(metrics.TaskSchedulerScope),

		notifyChan:   make(chan struct{}),
		shutdownChan: make(chan struct{}),

		priorityToWeight: priorityToWeight,
	}
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) Start() {
	if !atomic.CompareAndSwapInt32(
		&s.status,
		common.DaemonStatusInitialized,
		common.DaemonStatusStarted,
	) {
		return
	}

	s.processor.Start()

	go s.eventLoop()

	s.logger.Info("interleaved weighted round robin task scheduler started")
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) Stop() {
	if !atomic.CompareAndSwapInt32(
		&s.status,
		common.DaemonStatusStarted,
		common.DaemonStatusStopped,
	) {
		return
	}

	close(s.shutdownChan)

	s.processor.Stop()

	s.logger.Info("interleaved weighted round robin task scheduler stopped")
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) Submit(
	task PriorityTask,

) {
	channel := s.getOrCreateTaskChan(task.GetPriority())
	channel.Chan() <- task
	s.notifyDispatcher()
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) eventLoop() {
	for {
		select {
		case <-s.notifyChan:
			s.dispatch()
		case <-s.shutdownChan:
			return
		}
	}
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) getOrCreateTaskChan(
	priority int,
) *WeightedChan {

}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) dispatch() {
	weightedChans := s.sortTaskChans()
	s.dispatchTaskChans(weightedChans)
	s.notifyRamainingTasks()
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) sortTaskChans() []*WeightedChan {
	s.RLock()
	weightedChans := make(WeightedChans, len(s.weightToTaskChans))
	if len(weightedChans) == 0 {
		s.RUnlock()
		return weightedChans
	}

	for _, weightedChan := range s.weightToTaskChans {
		weightedChans = append(weightedChans, weightedChan)
	}
	s.RUnlock()
	sort.Sort(weightedChans)
	return weightedChans
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) dispatchTaskChans(
	weightedChans []*WeightedChan,
) {
	dispatchChans := make([]*WeightedChan, 0, weightedChans)
	index := len(weightedChans) - 1
	maxWeight := weightedChans[index].Weight()

	for round := maxWeight - 1; round > -1; round-- {
		for ; index >= 0 && weightedChans[index].Weight() > round; index-- {
			dispatchChans = append(dispatchChans, weightedChans[index])
		}

	LoopDispatch:
		for _, dispatchChan := range dispatchChans {
			select {
			case task := <-dispatchChan:
				s.processor.Submit(task)
			case <-s.shutdownChan:
				return
			default:
				continue LoopDispatch
			}
		}
	}
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) notifyRamainingTasks() {
	s.RLock()
	defer s.RUnlock()

	for _, weightedChan := range s.weightToTaskChans {
		if weightedChan.Len() > 0 {
			s.notifyDispatcher()
		}
	}
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) notifyDispatcher() {
	if s.isStopped() {
		s.drainTasks()
		return
	}

	select {
	case s.notifyChan <- struct{}{}:
	default:
	}
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) drainTasks() {
	for _, channel := range s.sortTaskChans() {
		for {
			select {
			case task := <-channel:
				task.Reschedule()
			default:
			}
		}
	}
}

func (s *InterleavedWeightedRoundRobinSchedulerImpl) isStopped() bool {
	return atomic.LoadInt32(&s.status) == common.DaemonStatusStopped
}
