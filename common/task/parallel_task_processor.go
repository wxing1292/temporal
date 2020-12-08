package task

import (
	"sync/atomic"

	"go.temporal.io/server/common"
	"go.temporal.io/server/common/backoff"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
)

type (
	// ParallelTaskProcessorOptions configs FIFO task scheduler
	ParallelTaskProcessorOptions struct {
		QueueSize   int
		WorkerCount int
		RetryPolicy backoff.RetryPolicy
	}

	ParallelTaskProcessorImpl struct {
		options *ParallelTaskProcessorOptions

		status       int32
		logger       log.Logger
		metricsScope metrics.Scope

		tasksChan    chan Task
		shutdownChan chan struct{}
	}
)

// NewParallelTaskProcessor creates a new PriorityTaskProcessor
func NewParallelTaskProcessor(
	logger log.Logger,
	metricsClient metrics.Client,
	options *ParallelTaskProcessorOptions,
) Processor {
	return &ParallelTaskProcessorImpl{
		options: options,

		status:       common.DaemonStatusInitialized,
		logger:       logger,
		metricsScope: metricsClient.Scope(metrics.ParallelTaskProcessingScope),

		tasksChan:    make(chan Task, options.QueueSize),
		shutdownChan: make(chan struct{}),
	}
}

func (p *ParallelTaskProcessorImpl) Start() {
	if !atomic.CompareAndSwapInt32(
		&p.status,
		common.DaemonStatusInitialized,
		common.DaemonStatusStarted,
	) {
		return
	}

	for i := 0; i < p.options.WorkerCount; i++ {
		go p.processTask()
	}

	p.logger.Info("parallel task processor started")
}

func (p *ParallelTaskProcessorImpl) Stop() {
	if !atomic.CompareAndSwapInt32(
		&p.status,
		common.DaemonStatusStarted,
		common.DaemonStatusStopped,
	) {
		return
	}

	close(p.shutdownChan)

	p.logger.Info("parallel task processor stopped")
}

func (p *ParallelTaskProcessorImpl) Submit(
	task Task,
) {
	p.metricsScope.IncCounter(metrics.ParallelTaskSubmitRequest)
	sw := p.metricsScope.StartTimer(metrics.ParallelTaskSubmitLatency)
	defer sw.Stop()

	p.tasksChan <- task
	if p.isStopped() {
		p.drainTasks()
	}
}

func (p *ParallelTaskProcessorImpl) processTask() {
	for {
		select {
		case task := <-p.tasksChan:
			p.executeTask(task)
		case <-p.shutdownChan:
			return
		}
	}
}

func (p *ParallelTaskProcessorImpl) executeTask(
	task Task,
) {
	stopWatch := p.metricsScope.StartTimer(metrics.ParallelTaskTaskProcessingLatency)
	defer stopWatch.Stop()

	operation := func() error {
		if err := task.Execute(); err != nil {
			return task.HandleErr(err)
		}
		return nil
	}

	isRetryable := func(err error) bool {
		return !p.isStopped() && task.RetryErr(err)
	}

	err := backoff.Retry(
		operation,
		p.options.RetryPolicy,
		isRetryable,
	)
	if err != nil {
		if p.isStopped() {
			task.Reschedule()
			return
		}

		task.Nack()
		return
	}

	task.Ack()
}

func (p *ParallelTaskProcessorImpl) drainTasks() {
LoopDrain:
	for {
		select {
		case task := <-p.tasksChan:
			task.Reschedule()
		default:
			break LoopDrain
		}
	}
}

func (p *ParallelTaskProcessorImpl) isStopped() bool {
	return atomic.LoadInt32(&p.status) == common.DaemonStatusStopped
}
