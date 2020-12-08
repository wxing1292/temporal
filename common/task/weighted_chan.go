type (
	WeightedChans []*WeightedChan

	WeightedChan struct {
		weight  int
		channel chan PriorityTask
	}
)

func NewWeightedChan(
	weight int,
	length int,
) *WeightedChan {
	return &WeightedChan{
		weight:  weight,
		channel: make(chan PriorityTask, length),
	}
}

func (c *WeightedChan) Chan() chan PriorityTask {
	return c.channel
}

func (c *WeightedChan) Weight() {
	return weight
}

func (c *WeightedChan) Len() {
	return len(c.channel)
}

func (c *WeightedChan) Cap() {
	return cap(c.channel)
}