package priqueue

import "container/heap"

type item struct {
	value    interface{}
	priority int
}
type sstack struct {
	size      int
	capacity  int
	offset    int
	blocksize int
	blocks    [][]*item
	active    []*item
}

func newSStack(size int) *sstack {
	result := new(sstack)
	result.active = make([]*item, size)
	result.blocks = [][]*item{result.active}
	result.capacity = size
	result.blocksize = size
	return result
}

func (s *sstack) Push(data interface{}) {
	if s.size == s.capacity {
		s.active = make([]*item, s.blocksize)
		s.blocks = append(s.blocks, s.active)
		s.capacity += s.blocksize
		s.offset = 0
	} else if s.offset == s.blocksize {
		s.active = s.blocks[s.size/s.blocksize]
		s.offset = 0
	}
	s.active[s.offset] = data.(*item)
	s.offset++
	s.size++
}

func (s *sstack) Pop() (res interface{}) {
	s.size--
	s.offset--
	if s.offset < 0 {
		s.offset = s.blocksize - 1
		s.active = s.blocks[s.size/s.blocksize]
	}
	res, s.active[s.offset] = s.active[s.offset], nil
	return
}

func (s *sstack) Len() int {
	return s.size
}

func (s *sstack) Less(i, j int) bool {
	return s.blocks[i/s.blocksize][i%s.blocksize].priority > s.blocks[j/s.blocksize][j%s.blocksize].priority
}

func (s *sstack) Swap(i, j int) {
	ib, io, jb, jo := i/s.blocksize, i%s.blocksize, j/s.blocksize, j%s.blocksize
	s.blocks[ib][io], s.blocks[jb][jo] = s.blocks[jb][jo], s.blocks[ib][io]
}
func (s *sstack) Reset() {
	*s = *newSStack(s.size)
}

type Priqueue struct {
	cont      *sstack
	blocksize int
}

func New(blocksize int) *Priqueue {
	return &Priqueue{
		cont:      newSStack(blocksize),
		blocksize: blocksize,
	}
}

func (p *Priqueue) Push(data interface{}, priority int) {
	heap.Push(p.cont, &item{data, priority})
}

func (p *Priqueue) Pop() (interface{}, int) {
	item := heap.Pop(p.cont).(*item)
	return item.value, item.priority
}

func (p *Priqueue) PopItem() interface{} {
	return heap.Pop(p.cont).(*item).value
}

func (p *Priqueue) Empty() bool {
	return p.cont.Len() == 0
}

func (p *Priqueue) Size() int {
	return p.cont.Len()
}

func (p *Priqueue) Reset() {
	*p = *New(p.blocksize)
}
