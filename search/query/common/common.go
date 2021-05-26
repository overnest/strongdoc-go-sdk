package common

type QueryOp string

const (
	Phrase   QueryOp = "phrase"
	TermAnd  QueryOp = "termand"
	TermOr   QueryOp = "termor"
	QueryAnd QueryOp = "queryand"
	QueryOr  QueryOp = "queryor"
)
