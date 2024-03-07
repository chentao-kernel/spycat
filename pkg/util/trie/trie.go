package trie

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"sort"
	"sync"

	"github.com/chentao-kernel/spycat/pkg/util/varint"
)

type trieNode struct {
	name     []byte
	value    uint64
	children []*trieNode
}

func mergeFunc(a, b uint64, t1, t2 *Trie) uint64 {
	return a*uint64(t1.Multiplier)/uint64(t1.Divider) + b*uint64(t2.Multiplier)/uint64(t2.Divider)
}

// func MergeSumVarintsWithWeights(t1, t2 *Trie) mergeFunc {
// 	f := func(a, b uint64) uint64 {
// 		return a*uint64(t1.Multiplier)/uint64(t1.Divider) + b*uint64(t2.Multiplier)/uint64(t2.Divider)
// 	}
// 	return f
// }

func newTrieNode(name []byte) *trieNode {
	return &trieNode{
		name:     name,
		children: make([]*trieNode, 0),
	}
}

func (tn *trieNode) clone() *trieNode {
	newTn := &trieNode{
		name:     tn.name,
		value:    tn.value,
		children: make([]*trieNode, len(tn.children)),
	}

	for i, c := range tn.children {
		newTn.children[i] = c.clone()
	}

	return newTn
}

func (tn *trieNode) insert(t2 *trieNode) {
	key := t2.name
	i := sort.Search(len(tn.children), func(i int) bool { return bytes.Compare(tn.children[i].name, key) >= 0 })

	tn.children = append(tn.children, &trieNode{})
	copy(tn.children[i+1:], tn.children[i:])
	tn.children[i] = t2
}

// TODO: Refactor
func (tn *trieNode) findNodeAt(key []byte, fn func(*trieNode)) {
	key2 := make([]byte, len(key))
	// TODO: remove
	copy(key2, key)
	key = key2
OuterLoop:
	for {
		// log.Debug("findNodeAt, key", string(key))

		if len(key) == 0 {
			fn(tn)
			return
		}

		// 4 options:
		// trie:
		// foo -> bar
		// 1) no leads (baz)
		//    create a new child, call fn with it
		// 2) lead, key matches (foo)
		//    call fn with existing child
		// 3) lead, key matches, shorter (fo / fop)
		//    split existing child, set that as tn
		// 4) lead, key matches, longer (fooo)
		//    go to existing child, set that as tn

		leadIndex := -1
		for k, v := range tn.children {
			if v.name[0] == key[0] {
				leadIndex = k
			}
		}

		if leadIndex == -1 { // 1
			// log.Debug("case 1")
			newTn := newTrieNode(key)
			tn.insert(newTn)
			fn(newTn)
			return
		}

		leadKey := tn.children[leadIndex].name
		// log.Debug("lead key", string(leadKey))
		lk := len(key)
		llk := len(leadKey)
		for i := 0; i < lk; i++ {
			if i == llk { // 4 fooo / foo i = 3 llk = 3
				// log.Debug("case 4")
				tn = tn.children[leadIndex]
				key = key[llk:]
				continue OuterLoop
			}
			if leadKey[i] != key[i] { // 3
				// log.Debug("case 3")
				// leadKey = abc
				// key = abd
				a := leadKey[:i] // ab
				b := leadKey[i:] // c
				// log.Debug("children ", string(a), string(b))
				// tn.childrenKeys[leadIndex] = a
				newTn := newTrieNode(a)
				// newTn.childrenKeys = [][]byte{b}
				newTn.children = []*trieNode{tn.children[leadIndex]}
				tn.children[leadIndex].name = b
				tn.children[leadIndex] = newTn
				// newTn.value = tn.value
				// tn.value = nil
				tn = newTn
				key = key[i:]
				continue OuterLoop
			}
		}
		// lk < llk
		if !bytes.Equal(key, leadKey) { // 3
			// log.Debug("case 3.2")
			a := leadKey[:lk] // ab
			b := leadKey[lk:] // c
			// tn.childrenKeys[leadIndex] = a
			newTn := newTrieNode(a)
			// newTn.childrenKeys = [][]byte{b}
			newTn.children = []*trieNode{tn.children[leadIndex]}
			tn.children[leadIndex].name = b
			tn.children[leadIndex] = newTn
			tn = newTn
			key = key[lk:]
			continue OuterLoop
		}

		// 2
		// log.Debug("case 2 —", lk, llk, bytes.Equal(key, leadKey), string(key), string(leadKey))
		fn(tn.children[leadIndex])
		return
	}
}

type Trie struct {
	mutex      sync.Mutex
	I          byte // debugging
	metadata   map[string]string
	Multiplier int
	Divider    int
	root       *trieNode
}

// New returns a new initialized empty Trie.
func New() *Trie {
	return &Trie{
		metadata:   make(map[string]string),
		root:       newTrieNode([]byte{}),
		Multiplier: 1,
		Divider:    1,
	}
}

func (t *Trie) Clone(m, d int) *Trie {
	return &Trie{
		I:          t.I,
		metadata:   t.metadata,
		Multiplier: m,
		Divider:    d,
		root:       t.root,
	}
}

func (t *Trie) Insert(key []byte, value uint64, merge ...bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	isMerge := false
	if len(merge) > 0 && merge[0] {
		isMerge = true
	}
	if isMerge {
		t.root.findNodeAt(key, func(tn *trieNode) {
			tn.value += value
		})
	} else {
		t.root.findNodeAt(key, func(tn *trieNode) {
			tn.value = value
		})
	}
}

func (t *Trie) Iterate(cb func(name []byte, val uint64)) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	nodes := []*trieNode{t.root}
	prefixes := make([][]byte, 1)
	prefixes[0] = make([]byte, 0)
	for len(nodes) > 0 {
		tn := nodes[0]
		nodes = nodes[1:]

		prefix := prefixes[0]
		prefixes = prefixes[1:]

		name := append(prefix, tn.name...)
		if tn.value > 0 {
			cb(name, tn.value)
		}
		// log.Debug("name", bytes.Index(name, []byte("\n")))

		nodes = append(tn.children, nodes...)
		for i := 0; i < len(tn.children); i++ {
			prefixes = append([][]byte{name}, prefixes...)
		}
	}
}

func (t *Trie) IsEmpty() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return len(t.root.children) == 0
}

func (t *Trie) Serialize(w io.Writer) error {
	nodes := []*trieNode{t.root}
	for len(nodes) > 0 {
		tn := nodes[0]
		nodes = nodes[1:]

		name := tn.name
		_, err := varint.Write(w, uint64(len(name)))
		if err != nil {
			return err
		}
		_, err = w.Write(name)
		if err != nil {
			return err
		}

		val := tn.value
		if t.Divider != 1 || t.Multiplier != 1 {
			val = val * uint64(t.Multiplier) / uint64(t.Divider)
		}
		_, err = varint.Write(w, uint64(val))
		if err != nil {
			return err
		}
		_, err = varint.Write(w, uint64(len(tn.children)))
		if err != nil {
			return err
		}

		nodes = append(tn.children, nodes...)
	}
	return nil
}

type offset struct {
	descCount int
	suffixLen int
}

// IterateRaw iterates through the serialized trie and calls cb function for
// every leaf. k references bytes from buf, therefore it must not be modified
// or used outside of cb, a copy of k should be used instead.
func IterateRaw(r io.Reader, buf []byte, cb func(k []byte, v int)) error {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

	b := bytes.NewBuffer(buf)
	var offsets []offset
	var copied int64
	for {
		nameLen, err := varint.Read(br)
		switch {
		case err == nil:
		case errors.Is(err, io.EOF):
			return nil
		default:
			return err
		}
		if nameLen != 0 {
			copied, err = b.ReadFrom(io.LimitReader(br, int64(nameLen)))
			if err != nil {
				return err
			}
		}
		value, err := varint.Read(br)
		if err != nil {
			return err
		}
		descCount, err := varint.Read(br)
		if err != nil {
			return err
		}

		// It may be a node or a leaf. Regardless, if it has
		// a value, there was a corresponding signature.
		if value > 0 {
			cb(b.Bytes(), int(value))
		}

		if descCount != 0 {
			// A node. Add node suffix and save offset.
			offsets = append(offsets, offset{
				descCount: int(descCount),
				suffixLen: int(copied),
			})
			continue
		}

		// A leaf. Cut the current label.
		b.Truncate(b.Len() - int(copied))
		// Cut parent suffix, if it has no more
		// descendants, and it is not the root.
		i := len(offsets) - 1
		if i < 0 {
			continue
		}
		offsets[i].descCount--
		for ; i > 0; i-- {
			if offsets[i].descCount != 0 {
				break
			}
			// No descending nodes left.
			// Cut suffix and remove the offset.
			b.Truncate(b.Len() - offsets[i].suffixLen)
			offsets = offsets[:i]
			// Decrease parent counter, if applicable.
			if p := len(offsets) - 1; p > 0 {
				offsets[p].descCount--
			}
		}
	}
}

func Deserialize(r io.Reader) (*Trie, error) {
	t := New()
	br := bufio.NewReader(r) // TODO if it's already a bytereader skip

	parents := []*trieNode{t.root}
	for len(parents) > 0 {
		parent := parents[0]
		parents = parents[1:]

		nameLen, err := varint.Read(br)
		// if err == io.EOF {
		//      return t, nil
		// }
		nameBuf := make([]byte, nameLen) // TODO: there are better ways to do this?
		_, err = io.ReadAtLeast(br, nameBuf, int(nameLen))
		// log.Debug(n, len(parents))
		// log.Debugf("%d", nameLen, string(nameBuf), n)
		if err != nil {
			return nil, err
		}
		tn := newTrieNode(nameBuf)
		// TODO: insert into parent
		parent.insert(tn)

		tn.value, err = varint.Read(br)
		if err != nil {
			return nil, err
		}

		childrenLen, err := varint.Read(br)
		if err != nil {
			return nil, err
		}

		for i := uint64(0); i < childrenLen; i++ {
			parents = append([]*trieNode{tn}, parents...)
		}
	}

	t.root = t.root.children[0]

	return t, nil
}

func (t *Trie) Bytes() []byte {
	b := bytes.Buffer{}
	t.Serialize(&b)
	return b.Bytes()
}

func FromBytes(p []byte) *Trie {
	t, _ := Deserialize(bytes.NewReader(p))
	return t
}
