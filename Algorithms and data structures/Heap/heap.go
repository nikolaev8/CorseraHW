package main

type heap struct {
	Data []float64
	swapCache [][]int // Counter of swap method calls
}

func NewHeap() *heap {
	data := make([]float64, 0)
	swapCache := make([][]int, 0)
	return &heap{
		data,
		swapCache,
	}
}

// nodeExists method check is the node with number vNum exist
func (h *heap) nodeExists(vNum int) bool {
	if ! (vNum >= 0 && vNum < len(h.Data) ) {
		return false
	}
	return true
}

// Parent method returns the number of the parent node for node with number vNum
// Return 0 if there is no node with number vNum in heap
func (h *heap) Parent(vNum int) int {
	if h.nodeExists(vNum) {
		return (vNum - 1) / 2
	}
	return 0
}

// LeftChild method returns the number of the left child node for node with number vNum
// Returns 0 if node vNum has no left child
func (h *heap) LeftChild(vNum int) int {
	childVNum := 2 * vNum + 1

	if h.nodeExists(vNum) {
		if h.nodeExists(childVNum) {
			return childVNum
		}
	}
	return 0
}

// RightChild method returns the number of the right child node for node with number vNum
// Returns 0 if node vNum has no right child
func (h *heap) RightChild(vNum int) int {
	childVNum := 2 * vNum + 2

	if h.nodeExists(vNum) {
		if h.nodeExists(childVNum) {
			return childVNum
		}
	}
	return 0
}

func (h *heap) swap (i, j int) {
	if h.nodeExists(i) && h.nodeExists(j) {
		h.Data[i], h.Data[j] = h.Data[j], h.Data[i]
		h.swapCache = append(h.swapCache, []int{i,j})
	}
}

// SiftUp method sifts up node with number vNum
func (h *heap) SiftUp(vNum int) {

	if !h.nodeExists(vNum) {
		return
	}
	for vNum > 0 && h.Data[h.Parent(vNum)] >  h.Data[vNum] {
		h.swap(h.Parent(vNum), vNum)
		vNum = h.Parent(vNum)
	}
}

func (h *heap) isLeaf(vNum int) bool{
	if ! (h.RightChild(vNum) == 0 && h.LeftChild(vNum) == 0){
		//fmt.Println("Node", vNum, "is not a leaf")
		return false
	}
	//fmt.Println("Node", vNum, "is a leaf")
	return true
}

// hasGoodChild check is node vNum has child with correct value for min-heap
func (h *heap) hasGoodChild(vNum int) bool {
	if !h.nodeExists(vNum) {
		return false
	}

	if lc := h.LeftChild(vNum); lc != 0 {
		if h.Data[vNum] > h.Data[lc] {
			return false
		}
	}

	if rc := h.RightChild(vNum); rc != 0 {
		if h.Data[vNum] > h.Data[rc] {
			return false
		}
	}

	return true
}

// SiftDown method sifts down node with number vNum
func (h *heap) SiftDown(vNum int) {

	if !h.nodeExists(vNum) {
		return
	}

	for !h.isLeaf(vNum) && !h.hasGoodChild(vNum) {

		lc := h.LeftChild(vNum)
		rc := h.RightChild(vNum)

		if lc == 0 {
			if h.Data[vNum] > h.Data[rc] {
				h.swap(vNum, rc)
				vNum = rc
				continue
			}
		}

		if rc == 0 {
			if h.Data[vNum] > h.Data[lc] {
				h.swap(vNum, lc)
				vNum = lc
				continue
			}
		}

		if h.Data[lc] > h.Data[rc] {
			h.swap(vNum, rc)
			vNum = rc
		} else {
			h.swap(vNum, lc)
			vNum = lc
		}
	}
}

// Insert method inserts node vNu with priority p to heap
func (h *heap) Insert(p float64) int {
	vNum := len(h.Data)
	h.Data = append(h.Data, p)
	h.SiftUp(vNum)
	return vNum
}

func (h *heap) ExtractMin() float64 {
	minVal := h.Data[0]
	lastNode := len(h.Data)-1
	h.Data[0] = h.Data[lastNode]
	h.Data = h.Data[:lastNode]
	h.SiftDown(0)
	return minVal
}

func (h *heap) Remove(vNum int) {
	if !h.nodeExists(vNum) {
		return
	}

	h.Data[vNum] = h.Data[0]-1
	h.SiftUp(vNum)
	_ = h.ExtractMin()
}

func (h *heap) ChangePriority(vNum int, p float64) {
	if !h.nodeExists(vNum) {
		return
	}

	oldP := h.Data[vNum]
	h.Data[vNum] = p
	if p > oldP {
		h.SiftDown(vNum)
	} else {
		h.SiftUp(vNum)
	}
	return
}

func BuildHeap(pArr []float64) (*heap, [][]int){
	h := &heap{
		Data: pArr,
	}

	for i:=(len(pArr) - 1 )/2; i>=0; i-=1 {
		h.SiftDown(i)
	}

	cache := h.swapCache
	h.swapCache = [][]int{}
	return h, cache
}

