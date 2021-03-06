package tbp

import (
	"regexp"
	"strings"
)

// Block defines the []string txt block receiver of search functions
type Block []string

// MatchInBlock to match multi-word in one line and multi-line in one block
// returns false, nil if none of the lines matches the pattern,
// if match, it returns true and [][]string for every submatched strings
// for each matched line.
// Empty submatch still take space as "", and result won't be nil.
func (b *Block) MatchInBlock(p *regexp.Regexp) (m bool, r [][]string) {
	for _, line := range *b {
		l := p.FindStringSubmatch(line)
		if l != nil {
			r = append(r, l[1:])
			m = true
		}
	}
	return
}

// SoloMatchInBlock to match a single value in whole block
// returns false, nil if no matche,
// if match, it returns true and a string.
// Empty submatch still take space as "", and result won't be nil.
func (b *Block) SoloMatchInBlock(p *regexp.Regexp) (m bool, r string) {
	m, mv := b.MatchInBlock(p)
	if !m {
		return
	}
	return m, mv[0][0]
}

// SliceMatchInBlock to match a single value in one line but multi lines in whole block
// returns false, nil if no matche,
// if match, it returns true and a []string.
// Empty submatch still take space as "", and result won't be nil.
func (b *Block) SliceMatchInBlock(p *regexp.Regexp) (m bool, r []string) {
	m, mv := b.MatchInBlock(p)
	if !m {
		return
	}
	for _, l := range mv {
		r = append(r, l[0])
	}
	return
}

// RemoveFromBlock returns the reference of a new block compiled with all unmatched lines
// return false if none of the lines are matched
func (b *Block) RemoveFromBlock(p *regexp.Regexp) (bool, *Block) {
	var nb Block
	var m bool
	for _, line := range *b {
		l := p.FindStringSubmatch(line)
		if l != nil {
			// matched line
			m = true
		} else {
			nb = append(nb, line)
		}
	}
	return m, &nb
}

/*
FetchBlock uses start and end patterns to find and return a slice of sub TxtBlocks.
The line matched by end pattern is not included in the block, but may included in
the next block only if it is the start line of that block.
If end patter is not specified, it will be created by block sub-pattern
defined as part of start pattern.
Blocks is nil if there is not a single matched block.
titleCatch is a convenient way to catch sub-match strings in the blockstart line.
Index of titleCatch is aligned with blocks.
sample pattern: `^(.*?)([A-Z]\S+)\s+(Up|Down)\s+(Up|Down)\S*\s+(\S+)\s+(\S+)$`
*/
func (b *Block) FetchBlock(s *regexp.Regexp, e *regexp.Regexp) (blocks []*Block, titleCatch [][]string) {
	inBlock := false // flag to mark for search end pattern
	var block *Block
	for i, line := range *b {
		// skip empty or space only line
		if regexp.MustCompile(`^\s*$`).MatchString(line) {
			continue
		}
		if inBlock {
			// already found start line, looking for the end
			l := e.FindStringSubmatch(line)
			if l == nil {
				// not the end line
				// save this line
				*block = append(*block, line)
				// loop to the end, and no end pattern matched
				if i == len(*b)-1 {
					blocks = append(blocks, block)
				}
				continue
			} else {
				// found end line
				inBlock = false
				// save the block
				blocks = append(blocks, block)
				// this line could be the start line of next matched block
				// pass through the pipe
			}
		}
		// looking for start line
		l := s.FindStringSubmatch(line)
		// found start line
		if l != nil {
			inBlock = true
			// set end pattern
			if e == nil {
				// escape the special regex characters
				re := regexp.MustCompile(`([\.\^\$\*\+\?\{\}\[\]\|\(\)])`)
				escp := re.ReplaceAllStringFunc(l[1], func(subm string) string {
					return map[string]string{
						`.`: `\.`,
						`^`: `\^`,
						`$`: `\$`,
						`*`: `\*`,
						`+`: `\+`,
						`?`: `\?`,
						`{`: `\{`,
						`}`: `\}`,
						`[`: `\[`,
						`]`: `\]`,
						`(`: `\(`,
						`)`: `\)`,
						`|`: `\|`,
					}[subm]
				})
				e = regexp.MustCompile(`^` + escp + `\S`)
			}
			// save title catch
			if len(l) > 2 {
				titleCatch = append(titleCatch, l[2:])
			} else {
				titleCatch = append(titleCatch, nil)
			}
			// new txt block, block now point to thi new obj
			bv := Block{}
			block = &bv
			// save this line
			*block = append(*block, line)
		}
	}
	// set itleCatch as nil if no title matched at all
	for _, t := range titleCatch {
		if t != nil {
			return
		}
	}
	titleCatch = nil
	return
}

// cut separate the txt block to lst of sub blocks based on the start line pattern
// titlecatch in sync with return block list
// no match will return an empty list
func (b *Block) Cut(s *regexp.Regexp) (blocks []*Block, titleCatch [][]string) {
	pblocks := []Block{}
	// use start line pattern to segment the block
	pblock := Block{}
	for _, line := range *b {
		if s.MatchString(line) {
			pblocks = append(pblocks, pblock)
			pblock = Block{}
		}
		pblock = append(pblock, line)
	}
	pblocks = append(pblocks, pblock)
	// only save the segment contains the start line,
	for _, seg := range pblocks {
		m, title := seg.MatchInBlock(s)
		if m {
			blocks = append(blocks, seg.Copy())
			titleCatch = append(titleCatch, title[0])
		}
	}
	return
}

// Segment separate the txt block to lst of sub blocks based on the end line pattern
// only blocks have the start line pattern will be returned
// titlecatch in sync with return block list
// no match will return an empty list
func (b *Block) Segment(s *regexp.Regexp, e *regexp.Regexp) (blocks []*Block, titleCatch [][]string) {
	pblocks := []Block{}
	// use end line pattern to segment the block
	pblock := Block{}
	for _, line := range *b {
		pblock = append(pblock, line)
		if e.MatchString(line) {
			pblocks = append(pblocks, pblock)
			pblock = Block{}
			continue
		}
	}
	// may have a empty segment attached to the end of segment list
	pblocks = append(pblocks, pblock)
	// only save the segment contains the start line,
	for _, seg := range pblocks {
		m, title := seg.MatchInBlock(s)
		if m {
			blocks = append(blocks, seg.Copy())
			titleCatch = append(titleCatch, title[0])
		}
	}
	return
}

// Copy returns a reference of the block deep copy
func (b *Block) Copy() *Block {
	// deep copy
	res := Block{}
	for _, l := range *b {
		res = append(res, l)
	}
	return &res
}

// String converts Block content back to single string
func (b *Block) String() (s string) {
	for _, l := range *b {
		s = s + l + "\n"
	}
	return
}

// Trim removes the tailing white space and extra empty lines
// more than 1 continous empty lines will be removed
func (b *Block) Trim() {
	nb := []string{}
	lastEmpty := false
	for _, l := range *b {
		nl := strings.TrimRight(l, " \n\t\r")
		if strings.TrimSpace(nl) == "" {
			if lastEmpty {
				continue
			}
			nl = ""
			lastEmpty = true
		} else {
			lastEmpty = false
		}
		nb = append(nb, nl)
	}

	*b = Block(nb)
}

// RmPeriod check each line of block and substitute time period string with "####"
func (b *Block) RmPeriod() {
	nb := []string{}
	for _, l := range *b {
		re := regexp.MustCompile(`\W(\d+[dh]\d\d[hm]\d\d[ms]|\d+[wd]\d+[dh]|\d\d:\d\d:\d\d)\W?`)
		nb = append(nb, re.ReplaceAllString(l, "####"))
	}

	*b = Block(nb)
}

// DiffFormat format the Block to a Diff optimized string
func (b *Block) DiffFormat(p map[string]string) (s string) {
	lastEmpty := false
	for _, l := range *b {
		// trim
		nl := strings.TrimRight(l, " \n\t\r")
		if strings.TrimSpace(nl) == "" {
			if lastEmpty {
				continue
			}
			nl = ""
			lastEmpty = true
		} else {
			lastEmpty = false
			// remove line
			if p["rmTSLine"] != "" {
				if regexp.MustCompile(p["rmTSLine"]).MatchString(nl) {
					continue
				}
			}
			// substitute string
			if p["subDurStr"] != "" {
				nl = regexp.MustCompile(p["subDurStr"]).ReplaceAllString(nl, "####")
			}
		}
		// assembly string
		s = s + nl + "\n"
	}
	return
}
