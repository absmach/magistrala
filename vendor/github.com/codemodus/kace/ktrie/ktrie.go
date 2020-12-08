package ktrie

import "unicode"

// KNode ...
type KNode struct {
	val   rune
	end   bool
	links []*KNode
}

// NewKNode ...
func NewKNode(val rune) *KNode {
	return &KNode{
		val:   val,
		links: make([]*KNode, 0),
	}
}

// Add ...
func (n *KNode) Add(rs []rune) {
	cur := n

	for k, v := range rs {
		link := cur.linkByVal(v)

		if link == nil {
			link = NewKNode(v)
			cur.links = append(cur.links, link)
		}

		if k == len(rs)-1 {
			link.end = true
		}

		cur = link
	}
}

// Find ...
func (n *KNode) Find(rs []rune) bool {
	cur := n

	for _, v := range rs {
		cur = cur.linkByVal(v)

		if cur == nil {
			return false
		}
	}

	return cur.end
}

// FindAsUpper ...
func (n *KNode) FindAsUpper(rs []rune) bool {
	cur := n

	for _, v := range rs {
		cur = cur.linkByVal(unicode.ToUpper(v))

		if cur == nil {
			return false
		}
	}

	return cur.end
}

func (n *KNode) linkByVal(val rune) *KNode {
	for _, v := range n.links {
		if v.val == val {
			return v
		}
	}

	return nil
}

// KTrie ...
type KTrie struct {
	*KNode

	maxDepth int
	minDepth int
}

// NewKTrie ...
func NewKTrie(data map[string]bool) (*KTrie, error) {
	n := NewKNode(0)

	maxDepth := 0
	minDepth := 9001

	for k := range data {
		rs := []rune(k)
		l := len(rs)

		n.Add(rs)

		if l > maxDepth {
			maxDepth = l
		}
		if l < minDepth {
			minDepth = l
		}
	}

	t := &KTrie{
		maxDepth: maxDepth,
		minDepth: minDepth,
		KNode:    n,
	}

	return t, nil
}

// MaxDepth ...
func (t *KTrie) MaxDepth() int {
	return t.maxDepth
}

// MinDepth ...
func (t *KTrie) MinDepth() int {
	return t.minDepth
}
