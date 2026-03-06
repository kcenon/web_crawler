package frontier

import "container/heap"

// priorityQueue implements heap.Interface for URLEntry items.
// Lower Priority values are dequeued first. For equal priorities,
// earlier DiscoveredAt times come first (FIFO within same priority).
type priorityQueue []*URLEntry

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority < pq[j].Priority
	}
	return pq[i].DiscoveredAt.Before(pq[j].DiscoveredAt)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x any) {
	*pq = append(*pq, x.(*URLEntry))
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[:n-1]
	return item
}

// newPriorityQueue creates an empty priority queue.
func newPriorityQueue() *priorityQueue {
	pq := make(priorityQueue, 0)
	heap.Init(&pq)
	return &pq
}
