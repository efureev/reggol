package reggol

type BlockFn = func(string) string
type Block struct {
	Text string
	Fn   BlockFn
}

func NewBlock(text string, fn BlockFn) Block {
	return Block{Text: text, Fn: fn}
}

func (b *Block) Value() string {
	if b.Fn != nil {
		return (b.Fn)(b.Text)
	}

	return b.Text
}

type Blocks []Block

func (bb *Blocks) Add(value string) *Blocks {
	return bb.AddBlock(Block{Text: value})
}

func (bb *Blocks) AddBlock(block Block) *Blocks {
	*bb = append(*bb, block)

	return bb
}
