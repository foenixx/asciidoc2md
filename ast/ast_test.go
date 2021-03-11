package ast

var (
	_ Block = (*Link)(nil)
	_ Block = (*Header)(nil)
	_ Block = (*Document)(nil)
	_ Block = (*List)(nil)
	_ Block = (*Image)(nil)
	_ Block = (*HorLine)(nil)
	_ Block = (*Paragraph)(nil)
	_ Block = (*ContainerBlock)(nil)
	_ Block = (*BlockTitle)(nil)
	_ Block = (*SyntaxBlock)(nil)
	_ Block = (*InlineImage)(nil)
	_ Block = (*Text)(nil)
	_ Block = (*Admonition)(nil)
	_ Block = (*Table)(nil)
	_ Block = (*Bookmark)(nil)
)

