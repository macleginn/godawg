package wordgraph6

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"unicode/utf8"
)

type treenode struct {
	id         int
	val        rune // Unicode value.
	children   *treenode
	next       *treenode
	parents    []*treenode
	endofword  bool
	hash       [20]byte
	level      int
	height     int
	firstchild bool
}

func (t *treenode) String() string {
	return fmt.Sprint(string(t.val), " ", t.level)
}

type arraynode struct {
	val      rune
	children rune
	eol      bool // End-of-list marker.
}

func NewDAWG() *treenode {
	root := new(treenode)
	root.id = -1
	root.level = -1
	root.val = '∅'
	return root
}

type flatteningQueue []*treenode // Impelents Interface

func (fq flatteningQueue) Len() int { return len(fq) }

func (fq flatteningQueue) Less(i, j int) bool {
	return fq[i].level < fq[j].level
}

func (fq flatteningQueue) Swap(i, j int) {
	fq[i], fq[j] = fq[j], fq[i]
}

func (fq *flatteningQueue) Push(tn *treenode) {
	*fq = append(*fq, tn)
}

// The value is returned from the front.
// The queue must be fully populated
// and then sorted.
func (fq *flatteningQueue) Pop() *treenode {
	old := *fq
	returnVal := old[0]
	*fq = old[1:]
	return returnVal
}

func (t *treenode) Put(s string, id *int) {
	// TODO: add some sanity checks.
	if t.id == 0 {
		t.id = 0
		*id++
	}
	fchar, size := utf8.DecodeRuneInString(s)
	s = s[size:]
	if t.children == nil {
		child := new(treenode)
		child.id = *id
		*id++
		child.val = fchar
		child.level = -1
		child.firstchild = true
		child.parents = append(child.parents, t)
		// Only first children are eligible for replacement
		// so don't bother initialising for others.
		child.put(s, id)
		t.children = child
	} else {
		var child *treenode
		for child = t.children; child.next != nil; child = child.next {
			if child.val == fchar {
				child.put(s, id)
				return
			}
		}
		if child.val == fchar { // Check the last child.
			child.put(s, id)
			return
		} else {
			newchild := new(treenode)
			newchild.id = *id
			*id++
			newchild.val = fchar
			newchild.level = -1
			newchild.put(s, id)
			child.next = newchild
		}
	}
}

func (t *treenode) put(s string, id *int) {
	if len(s) == 0 {
		return
	}
	fchar, size := utf8.DecodeRuneInString(s)
	s = s[size:]
	if t.children == nil {
		child := new(treenode)
		child.id = *id
		*id++
		child.val = fchar
		child.level = -1
		child.firstchild = true
		child.parents = append(child.parents, t)
		child.put(s, id)
		t.children = child
	} else {
		var child *treenode
		for child = t.children; child.next != nil; child = child.next {
			if child.val == fchar {
				child.put(s, id)
				return
			}
		}
		if child.val == fchar { // Check the last child.
			child.put(s, id)
			return
		} else {
			newchild := new(treenode)
			newchild.id = *id
			*id++
			newchild.val = fchar
			newchild.level = -1
			newchild.put(s, id)
			child.next = newchild
		}
	}
}

func (t *treenode) Optimise() {
	fmt.Println("Computing levels")
	t.computeLevels(0)
	fmt.Println("Computing heights")
	t.computeHeights()
	fmt.Println("Computing hashes")
	t.computeHashes()
	// heightlevels := make(map[int][]*treenode)
	// t.populateHeightLevels(&heightlevels)
	// var levels []int
	// for key := range heightlevels {
	// 	levels = append(levels, key)
	// }
	// // maxHeight := max(levels)
	// for i := 0; i < maxHeight; i++ {
	// 	fmt.Println("Processing nodes of height", i)
	// 	processLevel(heightlevels[i])
	// }

	maxHeight := t.height // root node is the highest
	for j := maxHeight - 1; j >= 0; j-- {
		nodesOfHeightX := make(map[*treenode]bool) // We use map to add all nodes only once.
		t.collectNodesOfHeightX(&nodesOfHeightX, j)
		var nodesOfTheSameHeight []*treenode
		for key := range nodesOfHeightX {
			nodesOfTheSameHeight = append(nodesOfTheSameHeight, key)
		}
		fmt.Println("Processing nodes of height", j)
		processLevel(nodesOfTheSameHeight)
	}
}

func (t *treenode) collectNodesOfHeightX(n *map[*treenode]bool, height int) {
	if t.height == height {
		(*n)[t] = true
	} else if t.height > height {
		for child := t.children; child != nil; child = child.next {
			child.collectNodesOfHeightX(n, height)
		}
	}
}

func (t *treenode) computeLevels(level int) {
	t.level = level
	if t.children != nil {
		for child := t.children; child != nil; child = child.next {
			child.computeLevels(level + 1)
		}
	}
}

func processLevel(level []*treenode) {
	var firsts []*treenode
	var others []*treenode
	for _, el := range level {
		if el.firstchild {
			firsts = append(firsts, el)
		} else {
			others = append(others, el)
		}
	}
	for _, first := range firsts {
		spent := false
		for _, le := range others {
			if first.val == le.val && first.hash == le.hash && first.level == le.level {
				first.redirect(le)
				spent = true
				break
			}
		}
		if !spent {
			others = append(others, first)
		}
	}
}

func (t *treenode) redirect(other *treenode) {
	if t.parents == nil {
		panic("This node should have at least one parent")
	}
	for _, parent := range t.parents {
		parent.children = other
		other.parents = append(other.parents, parent)
	}
}

func (t *treenode) populateHeightLevels(hl *map[int][]*treenode) {
	(*hl)[t.height] = append((*hl)[t.height], t)
	if t.children != nil {
		for child := t.children; child != nil; child = child.next {
			child.populateHeightLevels(hl)
		}
	}
}

func (t *treenode) computeHashes() []byte {
	var data []byte
	if t.next != nil {
		data = append(data, (t.next.computeHashes())...)
	}
	if t.children != nil {
		data = append(data, (t.children.computeHashes())...)
	}
	data = append(data, []byte(string(t.val))...)
	t.hash = sha1.Sum(data)
	return data
}

func (t *treenode) computeHeights() {
	if t.children == nil {
		t.height = 0
	} else {
		var childrenHeights []int
		for child := t.children; child != nil; child = child.next {
			child.computeHeights()
			childrenHeights = append(childrenHeights, child.height)
		}
		t.height = 1 + max(childrenHeights)
	}
}

func max(arr []int) int {
	var max int = 0
	for _, value := range arr {
		if value > max {
			max = value
		}
	}
	return max
}

func (a arraynode) String() string {
	return fmt.Sprintf("{%s, %d, %t}", string(a.val), a.children, a.eol)
}

type outarray []arraynode

func (o outarray) Len() int {
	return len(o)
}

func (o outarray) String() string {
	var buffer bytes.Buffer
	for i, val := range o {
		buffer.WriteString(fmt.Sprint(i, val, "\n"))
	}
	return buffer.String()
}

func (t *treenode) Flatten() {
	var output outarray
	unfilledParents := make(map[*treenode]int)
	allocatedNodes := make(map[*treenode]bool)
	currentLen := len(output)
	for i := 0; ; i++ {
		t.addNodesOfLevelX(&output, i, &unfilledParents, &allocatedNodes)
		if len(output) == currentLen {
			break
		} else {
			currentLen = len(output)
		}
	}
	output.createDot()
	output.writeToFile()
}

func (o outarray) writeToFile() {
	outfile, err := os.Create("dawg_big.wg")
	if err != nil {
		log.Fatal(err)
	}
	defer outfile.Close()
	for _, el := range o {
		binary.Write(outfile, binary.LittleEndian, el.val)
		binary.Write(outfile, binary.LittleEndian, el.children)
		binary.Write(outfile, binary.LittleEndian, el.eol)
	}
}

func (t *treenode) addNodesOfLevelX(array *outarray, level int, up *map[*treenode]int, an *map[*treenode]bool) {
	if t.level == level {
		if _, found := (*an)[t]; !found {
			*array = append(*array, arraynode{val: t.val, eol: t.next == nil})
			(*up)[t] = len(*array) - 1
			(*an)[t] = true
			if t.parents != nil {
				for _, parent := range t.parents {
					if _, found := (*an)[parent]; found {
						(*array)[(*up)[parent]].children = rune(len(*array) - 1)
					}
				}
			}
		}
	} else {
		if t.children != nil && t.children.firstchild {
			for child := t.children; child != nil; child = child.next {
				child.addNodesOfLevelX(array, level, up, an)
			}
		}
	}
}

func length(t *treenode) int {
	if t.next == nil {
		return 1
	} else {
		return 1 + length(t.next)
	}
}

func (t *treenode) in(m map[*treenode]rune) bool {
	_, found := m[t]
	return found
}

func (t *treenode) populateQueue(fq *flatteningQueue, nodes *map[*treenode]bool) {
	if _, found := (*nodes)[t]; !found {
		fq.Push(t)
		(*nodes)[t] = true
		if t.children != nil {
			for child := t.children; child != nil; child = child.next {
				child.populateQueue(fq, nodes)
			}
		}
	}
}

func (t *treenode) CreateDot(filename string) {
	nodesMap := make(map[int]string)
	t.populateNodes(&nodesMap)
	edgesMap := make(map[int][]int)
	edgesInMap := make(map[string]bool)
	t.populateEdges(&edgesMap, &edgesInMap)
	outfile, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer outfile.Close()
	writer := bufio.NewWriter(outfile)
	writer.WriteString("digraph Tree {\n\trankdir=LR\n")
	for key, value := range nodesMap {
		writer.WriteString(fmt.Sprintf("\t%d [label=\"%s\"];\n", key, value))
	}
	for key, value := range edgesMap {
		for i, el := range value {
			if i == 0 {
				writer.WriteString(fmt.Sprintf("%d -> %d;\n", key, el))
			} else {
				writer.WriteString(fmt.Sprintf("%d -> %d [style = \"dotted\"];\n", key, el))
			}
		}
	}
	writer.WriteString("}\n")
	writer.Flush()
}

func (t *treenode) populateNodes(nm *map[int]string) {
	(*nm)[t.id] = fmt.Sprintf("%s", string(t.val))
	if t.children != nil {
		for child := t.children; child != nil; child = child.next {
			child.populateNodes(nm)
		}
	}
}

func (t *treenode) populateEdges(nm *map[int][]int, eim *map[string]bool) {
	if t.children != nil {
		for child := t.children; child != nil; child = child.next {
			edge := fmt.Sprintf("%d->%d", t.id, child.id)
			// if _, found := (*eim)[edge]; !found {
			(*nm)[t.id] = append((*nm)[t.id], child.id)
			(*eim)[edge] = true
			child.populateEdges(nm, eim)
			// }
		}
	}
}

func (o outarray) createDot() {
	// fmt.Println(len(o))
	nodes := make(map[int]string)
	for i := range o {
		nodes[i] = string(o[i].val)
	}
	edges := make(map[int][]rune)
	for i := range o {
		j := o[i].children
		// edges[i] = append(edges[i], j)

		// fmt.Println(j)
		if j != 0 {
			for !o[j].eol {
				edges[i] = append(edges[i], j)
				j++
			}
			edges[i] = append(edges[i], j)
		}
	}
	filename := "array6.dot"
	outfile, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer outfile.Close()
	writer := bufio.NewWriter(outfile)
	writer.WriteString("digraph Array {\n\trankdir=LR\n")
	for key, value := range nodes {
		writer.WriteString(fmt.Sprintf("\t%d [label=\"%s\"];\n", key, value))
	}
	for out, in := range edges {
		for _, el := range in {
			if el != 0 {
				writer.WriteString(fmt.Sprintf("%d -> %d;\n", out, el))
			}
		}
	}
	writer.WriteString("}\n")
	writer.Flush()
}
