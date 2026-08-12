package reggol

// BlockFn decorates a block's text.
type BlockFn = func(string) string

// Block is a short marker rendered ahead of the message.
type Block struct {
	Text string
	Fn   BlockFn
}

// NewBlock creates a block with an optional decorator.
func NewBlock(text string, fn BlockFn) Block {
	return Block{Text: text, Fn: fn}
}

// Value returns the block's rendered text.
func (b Block) Value() string {
	if b.Fn != nil {
		return b.Fn(b.Text)
	}

	return b.Text
}

// appendTo appends the block's rendered text, avoiding the intermediate string
// when no decorator is set.
func (b Block) appendTo(dst []byte) []byte {
	if b.Fn != nil {
		return append(dst, b.Fn(b.Text)...)
	}

	return append(dst, b.Text...)
}

// Blocks is an ordered set of blocks.
type Blocks []Block

// Add appends a plain-text block.
func (bb *Blocks) Add(text string) *Blocks {
	*bb = append(*bb, Block{Text: text})

	return bb
}

// AddBlock appends a block.
func (bb *Blocks) AddBlock(block Block) *Blocks {
	*bb = append(*bb, block)

	return bb
}
