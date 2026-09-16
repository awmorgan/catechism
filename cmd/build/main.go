package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"
)

type Section struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Heading  string `json:"heading"`
	HTML     string `json:"html"`
	ParaNums []int  `json:"paraNums"`
	MinP     int    `json:"minP"`
	MaxP     int    `json:"maxP"`
}

type TOCNode struct {
	ID       string     `json:"id"`
	SecNum   int        `json:"secNum"`
	Title    string     `json:"title"`
	Range    string     `json:"range"`
	IsPart   bool       `json:"isPart"`
	IsBranch bool       `json:"isBranch"`
	Children []*TOCNode `json:"children,omitempty"`
}

type PageDef struct {
	Index       int
	Filename    string
	Title       string
	StartID     string
	StartSecNum int
	Breadcrumbs []string
	Sections    []*Section
	ParaNums    []int
	MinP        int
	MaxP        int
	Prev        *PageDef
	Next        *PageDef
}

type SearchItem struct {
	P     int    `json:"p"`
	Title string `json:"title"`
	Path  string `json:"path"`
	Text  string `json:"text"`
}

type ReferenceData struct {
	Definitions map[string]string `json:"definitions"`
	Notes       []struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"notes"`
}

func main() {
	sourceFile := "data/source.html"
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		sourceFile = "index.html"
	}
	fmt.Printf("Reading source from %s...\n", sourceFile)
	contentBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", sourceFile, err)
	}
	content := string(contentBytes)

	// 1. Extract reference-data
	refRegex := regexp.MustCompile(`(?s)<script type="application/json"\s+id="reference-data">(.*?)</script>`)
	refMatch := refRegex.FindStringSubmatch(content)
	var refData ReferenceData
	if len(refMatch) > 1 {
		if err := json.Unmarshal([]byte(refMatch[1]), &refData); err != nil {
			log.Printf("Warning: Failed to parse reference-data: %v", err)
		} else {
			fmt.Printf("Loaded %d glossary definitions and %d notes.\n", len(refData.Definitions), len(refData.Notes))
			os.MkdirAll("data", 0755)
			os.WriteFile("data/glossary.json", []byte(refMatch[1]), 0644)
		}
	}

	// 2. Extract sections
	sectionPrefix := `<section class="chapter" id="`
	parts := strings.Split(content, sectionPrefix)
	h2Regex := regexp.MustCompile(`(?s)<h2[^>]*>(.*?)</h2>`)
	pRegex := regexp.MustCompile(`id="p-(\d+)"`)

	var sections []*Section
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		quoteIdx := strings.Index(part, `">`)
		if quoteIdx == -1 {
			continue
		}
		id := part[:quoteIdx]
		body := part[quoteIdx+2:]
		if i == len(parts)-1 {
			if mainClose := strings.Index(body, "</main>"); mainClose != -1 {
				body = body[:mainClose]
			}
		}

		heading := ""
		if h2m := h2Regex.FindStringSubmatch(body); len(h2m) > 1 {
			heading = cleanHTMLText(h2m[1])
		}

		pMatches := pRegex.FindAllStringSubmatch(body, -1)
		var pNums []int
		minP, maxP := 0, 0
		for _, pm := range pMatches {
			n, _ := strconv.Atoi(pm[1])
			pNums = append(pNums, n)
			if minP == 0 || n < minP {
				minP = n
			}
			if n > maxP {
				maxP = n
			}
		}

		sec := &Section{
			Index:    i - 1,
			ID:       id,
			Heading:  heading,
			HTML:     body,
			ParaNums: pNums,
			MinP:     minP,
			MaxP:     maxP,
		}
		sections = append(sections, sec)
	}

	// 3. Parse TOC
	tocStart := strings.Index(content, `<nav aria-label="Table of contents">`)
	tocEnd := strings.Index(content, `</details><section class="chapter"`)
	tocHTML := content[tocStart:tocEnd]
	doc, err := nethtml.Parse(strings.NewReader(tocHTML))
	if err != nil {
		log.Fatalf("Failed to parse TOC HTML: %v", err)
	}
	tocRoots := parseTOC(doc)

	// Collect page definitions using inherited ancestor start sections for logical page boundaries
	type RawPage struct {
		Title       string
		Range       string
		StartID     string
		StartSecNum int
		Breadcrumbs []string
	}

	var rawPages []*RawPage
	lastAssignedSec := -1

	var walkTOC func(n *TOCNode, trail []string, inheritedStartSec int)
	walkTOC = func(n *TOCNode, trail []string, inheritedStartSec int) {
		currentTrail := trail
		if n.Title != "" {
			currentTrail = append(currentTrail, n.Title)
		}

		startSec := inheritedStartSec
		if n.SecNum != -1 && (startSec == -1 || n.SecNum < startSec) && n.SecNum > lastAssignedSec {
			startSec = n.SecNum
		}

		hasChildBranch := false
		for _, ch := range n.Children {
			if ch.IsBranch {
				hasChildBranch = true
				break
			}
		}

		if n.IsBranch && !hasChildBranch {
			actualStart := startSec
			if actualStart == -1 || actualStart <= lastAssignedSec {
				actualStart = n.SecNum
			}
			if actualStart > lastAssignedSec {
				lastAssignedSec = actualStart
			}

			rawPages = append(rawPages, &RawPage{
				Title:       n.Title,
				Range:       n.Range,
				StartID:     n.ID,
				StartSecNum: actualStart,
				Breadcrumbs: currentTrail,
			})
			return
		}

		first := true
		for _, ch := range n.Children {
			if ch.IsBranch {
				if first {
					walkTOC(ch, currentTrail, startSec)
					first = false
				} else {
					walkTOC(ch, currentTrail, -1)
				}
			}
		}
	}

	for _, root := range tocRoots {
		walkTOC(root, nil, -1)
	}

	if len(rawPages) > 0 {
		rawPages[0].StartSecNum = 0
		rawPages[0].StartID = "s-0"
	}

	// Slicing sections into contiguous pages using actual slice indices
	secIndexByID := make(map[string]int)
	for idx, s := range sections {
		secIndexByID[s.ID] = idx
	}

	getSecIndex := func(secNum int, secID string) int {
		if idx, ok := secIndexByID[secID]; ok {
			return idx
		}
		formattedID := fmt.Sprintf("s-%d", secNum)
		if idx, ok := secIndexByID[formattedID]; ok {
			return idx
		}
		for i, s := range sections {
			num, _ := strconv.Atoi(strings.TrimPrefix(s.ID, "s-"))
			if num >= secNum {
				return i
			}
		}
		return len(sections)
	}

	var pages []*PageDef
	for i := 0; i < len(rawPages); i++ {
		startSec := getSecIndex(rawPages[i].StartSecNum, rawPages[i].StartID)
		endSec := len(sections)
		if i+1 < len(rawPages) {
			nextStart := getSecIndex(rawPages[i+1].StartSecNum, rawPages[i+1].StartID)
			if nextStart > startSec {
				endSec = nextStart
			}
		}

		var pageSecs []*Section
		var pageParas []int
		minP, maxP := 0, 0
		for s := startSec; s < endSec && s < len(sections); s++ {
			sec := sections[s]
			pageSecs = append(pageSecs, sec)
			pageParas = append(pageParas, sec.ParaNums...)
			if sec.MinP > 0 && (minP == 0 || sec.MinP < minP) {
				minP = sec.MinP
			}
			if sec.MaxP > maxP {
				maxP = sec.MaxP
			}
		}

		filename := fmt.Sprintf("%03d.html", i+1)
		page := &PageDef{
			Index:       i + 1,
			Filename:    filename,
			Title:       rawPages[i].Title,
			StartID:     rawPages[i].StartID,
			Breadcrumbs: rawPages[i].Breadcrumbs,
			Sections:    pageSecs,
			ParaNums:    pageParas,
			MinP:        minP,
			MaxP:        maxP,
		}
		pages = append(pages, page)
	}

	// Link Prev / Next
	for i := 0; i < len(pages); i++ {
		if i > 0 {
			pages[i].Prev = pages[i-1]
		}
		if i < len(pages)-1 {
			pages[i].Next = pages[i+1]
		}
	}

	// Verify paragraph coverage
	allFoundParas := make(map[int]bool)
	duplicateParas := make(map[int]int)
	for _, p := range pages {
		for _, pn := range p.ParaNums {
			if allFoundParas[pn] {
				duplicateParas[pn]++
			}
			allFoundParas[pn] = true
		}
	}

	fmt.Printf("Total Pages Generated: %d\n", len(pages))
	fmt.Printf("Unique Paragraphs covered: %d / 2865\n", len(allFoundParas))
	if len(duplicateParas) > 0 {
		fmt.Printf("WARNING: %d duplicate paragraphs detected!\n", len(duplicateParas))
	}

	// 5. Build search index & para-map
	paraMap := make(map[string]string)
	var searchIndex []SearchItem
	plainParaRegex := regexp.MustCompile(`(?s)<p id="p-(\d+)"[^>]*>(.*?)</p>`)
	tagStripRegex := regexp.MustCompile(`<[^>]+>`)

	for _, page := range pages {
		for _, sec := range page.Sections {
			pMatches := plainParaRegex.FindAllStringSubmatch(sec.HTML, -1)
			for _, pm := range pMatches {
				pNum, _ := strconv.Atoi(pm[1])
				paraMap[strconv.Itoa(pNum)] = "pages/" + page.Filename

				// Clean text
				text := tagStripRegex.ReplaceAllString(pm[2], " ")
				text = strings.Join(strings.Fields(html.UnescapeString(text)), " ")
				searchIndex = append(searchIndex, SearchItem{
					P:     pNum,
					Title: page.Title,
					Path:  "pages/" + page.Filename,
					Text:  text,
				})
			}
		}
	}

	os.MkdirAll("assets", 0755)
	paraMapJSON, _ := json.Marshal(paraMap)
	os.WriteFile("assets/para-map.json", paraMapJSON, 0644)
	fmt.Printf("Wrote assets/para-map.json with %d mapped paragraphs.\n", len(paraMap))

	searchIndexJSON, _ := json.Marshal(searchIndex)
	os.WriteFile("assets/search-index.json", searchIndexJSON, 0644)
	fmt.Printf("Wrote assets/search-index.json with %d indexed paragraphs.\n", len(searchIndex))

	// 6. Generate HTML files in pages/
	os.RemoveAll("pages")
	os.MkdirAll("pages", 0755)

	drawerHTML := renderTOCDrawer(pages)

	for _, page := range pages {
		pageHTML := renderPageHTML(page, drawerHTML)
		filePath := filepath.Join("pages", page.Filename)
		if err := os.WriteFile(filePath, []byte(pageHTML), 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", filePath, err)
		}
	}
	fmt.Printf("Successfully generated %d chapter pages in pages/\n", len(pages))

	// 7. Generate Landing Page index.html
	landingHTML := renderLandingPage(pages)
	if err := os.WriteFile("index.html", []byte(landingHTML), 0644); err != nil {
		log.Fatalf("Failed to write landing index.html: %v", err)
	}
	fmt.Println("Successfully generated modern landing page at index.html!")
}

func renderTOCDrawer(pages []*PageDef) string {
	var sb strings.Builder
	currentPart := ""

	for _, p := range pages {
		partName := ""
		if len(p.Breadcrumbs) > 0 {
			partName = p.Breadcrumbs[0]
		}
		if partName != currentPart && partName != "" {
			currentPart = partName
			sb.WriteString(fmt.Sprintf(`<div class="toc-part-header">%s</div>`, html.EscapeString(currentPart)))
		}

		rangeStr := ""
		if p.MinP > 0 && p.MaxP > 0 {
			rangeStr = fmt.Sprintf("¶ %d–%d", p.MinP, p.MaxP)
		}

		sb.WriteString(fmt.Sprintf(`
<a href="%s" class="toc-page-link" data-page="%s">
  <span class="toc-link-title">%s</span>
  <span class="toc-link-range">%s</span>
</a>`, p.Filename, p.Filename, html.EscapeString(p.Title), rangeStr))
	}
	return sb.String()
}

func renderPageHTML(page *PageDef, drawerHTML string) string {
	currentDrawer := strings.Replace(drawerHTML,
		fmt.Sprintf(`data-page="%s"`, page.Filename),
		fmt.Sprintf(`data-page="%s" class="toc-page-link current-page" aria-current="page"`, page.Filename), 1)

	var bcHTML strings.Builder
	bcHTML.WriteString(`<a href="../index.html">Catechism</a>`)
	for _, b := range page.Breadcrumbs {
		bcHTML.WriteString(` <span style="opacity:0.5">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span>%s</span>`, html.EscapeString(b)))
	}

	var navHTML strings.Builder
	navHTML.WriteString(`<nav class="page-navigation" aria-label="Chapter Navigation">`)
	if page.Prev != nil {
		prevRange := ""
		if page.Prev.MinP > 0 {
			prevRange = fmt.Sprintf("¶ %d–%d", page.Prev.MinP, page.Prev.MaxP)
		}
		navHTML.WriteString(fmt.Sprintf(`
  <a href="%s" class="nav-card prev" rel="prev">
    <span class="nav-label">← Previous</span>
    <span class="nav-title">%s</span>
    <span class="nav-range">%s</span>
  </a>`, page.Prev.Filename, html.EscapeString(page.Prev.Title), prevRange))
	} else {
		navHTML.WriteString(`<div></div>`)
	}

	if page.Next != nil {
		nextRange := ""
		if page.Next.MinP > 0 {
			nextRange = fmt.Sprintf("¶ %d–%d", page.Next.MinP, page.Next.MaxP)
		}
		navHTML.WriteString(fmt.Sprintf(`
  <a href="%s" class="nav-card next" rel="next">
    <span class="nav-label">Next →</span>
    <span class="nav-title">%s</span>
    <span class="nav-range">%s</span>
  </a>`, page.Next.Filename, html.EscapeString(page.Next.Title), nextRange))
	}
	navHTML.WriteString(`</nav>`)

	var bodyContent strings.Builder
	for _, s := range page.Sections {
		bodyContent.WriteString(fmt.Sprintf(`<section class="chapter" id="%s">`, s.ID))
		bodyContent.WriteString(s.HTML)
		bodyContent.WriteString(`</section>`)
	}

	rangeStr := ""
	if page.MinP > 0 && page.MaxP > 0 {
		rangeStr = fmt.Sprintf("¶ %d–%d", page.MinP, page.MaxP)
	}

	partBadge := ""
	if len(page.Breadcrumbs) > 0 {
		partBadge = page.Breadcrumbs[0]
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s — Catechism of the Catholic Church</title>
  <meta name="description" content="Catechism of the Catholic Church: %s (%s)">
  <link rel="stylesheet" href="../assets/style.css">
  <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>📖</text></svg>">
</head>
<body data-root="../">
  <header class="site-header">
    <div class="header-container">
      <div class="header-left">
        <button type="button" id="btn-open-drawer" class="btn-icon" aria-label="Open Table of Contents" title="Table of Contents">
          📖 <span>Contents</span>
        </button>
        <a href="../index.html" class="header-title" title="Return to Table of Contents">Catechism</a>
      </div>
      <div class="header-right">
        <button type="button" id="btn-open-search" class="btn-icon" aria-label="Search Catechism" title="Search (/)">
          🔍
        </button>
        <form id="jump-form" class="jump-form">
          <input type="text" inputmode="numeric" pattern="[0-9]*" id="jump-input" class="jump-input" placeholder="¶ 1–2865" aria-label="Jump to paragraph number">
          <button type="submit" class="jump-btn">Go</button>
        </form>
        <button type="button" id="btn-font-smaller" class="btn-icon" aria-label="Smaller text" title="Smaller font">A−</button>
        <button type="button" id="btn-font-larger" class="btn-icon" aria-label="Larger text" title="Larger font">A+</button>
        <button type="button" id="btn-theme-toggle" class="btn-icon" aria-label="Toggle theme" title="Toggle light/dark theme">🌙</button>
      </div>
    </div>
  </header>

  <div class="breadcrumbs-bar">
    <div class="breadcrumbs-container">
      %s
    </div>
  </div>

  <main class="reader-main">
    <div class="chapter-title-group">
      <span class="chapter-badge">%s</span>
      <h1 class="chapter-h1">%s</h1>
      <span class="chapter-range">%s</span>
    </div>

    %s

    %s
  </main>

  <div id="drawer-backdrop" class="drawer-backdrop"></div>
  <aside id="toc-drawer" class="toc-drawer" aria-hidden="true">
    <div class="drawer-header">
      <h2>Table of Contents</h2>
      <button type="button" id="btn-close-drawer" class="btn-icon" aria-label="Close Table of Contents">✕</button>
    </div>
    <div class="drawer-content">
      %s
    </div>
  </aside>

  <dialog id="search-modal" class="modal-dialog">
    <div class="search-header">
      <span style="font-size:1.2rem">🔍</span>
      <input type="search" id="search-query" placeholder="Search the Catechism..." autocomplete="off">
      <button type="button" id="btn-close-search" class="btn-icon">✕</button>
    </div>
    <div class="search-controls-bar">
      <label class="search-option-label">
        <input type="checkbox" id="search-whole-words" checked>
        <span>Whole words</span>
      </label>
      <span id="search-count" class="search-count"></span>
    </div>
    <div id="search-results" class="search-results"></div>
  </dialog>

  <script src="../assets/reader.js"></script>
  <script src="../assets/search.js"></script>
</body>
</html>`,
		html.EscapeString(page.Title),
		html.EscapeString(page.Title),
		rangeStr,
		bcHTML.String(),
		html.EscapeString(partBadge),
		html.EscapeString(page.Title),
		rangeStr,
		bodyContent.String(),
		navHTML.String(),
		currentDrawer,
	)
}

func renderLandingPage(pages []*PageDef) string {
	var tocListHTML strings.Builder

	currentPart := ""
	for _, p := range pages {
		partName := ""
		if len(p.Breadcrumbs) > 0 {
			partName = p.Breadcrumbs[0]
		}
		if partName != currentPart && partName != "" {
			currentPart = partName
			tocListHTML.WriteString(fmt.Sprintf(`
<div style="margin: 2rem 0 0.6rem; padding-bottom: 0.4rem; border-bottom: 2px solid var(--accent); font-family: var(--font-sans); font-size: 1.15rem; font-weight: 700; color: var(--accent);">
  %s
</div>`, html.EscapeString(currentPart)))
		}

		rangeStr := ""
		if p.MinP > 0 && p.MaxP > 0 {
			rangeStr = fmt.Sprintf("¶ %d–%d", p.MinP, p.MaxP)
		}

		tocListHTML.WriteString(fmt.Sprintf(`
<a href="pages/%s" class="toc-page-link" style="background: var(--bg-card); border: 1px solid var(--border); padding: 0.75rem 1rem; margin-bottom: 0.35rem; border-radius: 8px;">
  <span class="toc-link-title" style="font-size: 0.95rem; font-weight: 500;">%s</span>
  <span class="toc-link-range" style="font-size: 0.82rem; background: var(--accent-light); color: var(--accent); padding: 0.15rem 0.45rem; border-radius: 4px;">%s</span>
</a>`, p.Filename, html.EscapeString(p.Title), rangeStr))
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Catechism of the Catholic Church — Modern Reader</title>
  <meta name="description" content="A fast, reader-friendly, and mobile-optimized edition of the Catechism of the Catholic Church with dark mode and adjustable typography.">
  <link rel="stylesheet" href="assets/style.css">
  <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>📖</text></svg>">
</head>
<body data-root="">
  <header class="site-header">
    <div class="header-container">
      <div class="header-left">
        <a href="index.html" class="header-title" style="font-size: 1.1rem; font-weight: 700;">Catechism</a>
      </div>
      <div class="header-right">
        <button type="button" id="btn-open-search" class="btn-icon" aria-label="Search Catechism" title="Search (/)">
          🔍 <span>Search</span>
        </button>
        <button type="button" id="btn-font-smaller" class="btn-icon" aria-label="Smaller text">A−</button>
        <button type="button" id="btn-font-larger" class="btn-icon" aria-label="Larger text">A+</button>
        <button type="button" id="btn-theme-toggle" class="btn-icon" aria-label="Toggle theme">🌙</button>
      </div>
    </div>
  </header>

  <main class="reader-main" style="max-width: 820px;">
    <div style="text-align: center; margin: 2rem 0 3rem; padding-bottom: 2rem; border-bottom: 1px solid var(--border);">
      <h1 style="font-size: 2.6rem; font-weight: 700; line-height: 1.2; margin-bottom: 1.5rem;">Catechism of the Catholic Church</h1>
      
      <div style="margin-top: 1rem; display: flex; justify-content: center; gap: 0.8rem; flex-wrap: wrap;">
        <a href="pages/001.html" class="btn-icon" style="height: 48px; padding: 0 1.5rem; background: var(--accent); color: #fff; text-decoration: none; font-size: 1.05rem; font-weight: 600; border: none;">
          📖 Begin Reading (Prologue)
        </a>
        <form id="jump-form" class="jump-form">
          <input type="text" inputmode="numeric" pattern="[0-9]*" id="jump-input" class="jump-input" placeholder="Go to ¶ (1–2865)" style="height: 48px; width: 8.5rem; font-size: 1rem;">
          <button type="submit" class="jump-btn" style="height: 48px; padding: 0 1rem; font-size: 1rem; font-weight: 600;">Go</button>
        </form>
      </div>
    </div>

    <section>
      <h2 style="font-family: var(--font-sans); font-size: 1.5rem; margin-bottom: 1rem; border: none; padding: 0;">Table of Contents</h2>
      <div style="display: flex; flex-direction: column;">
        %s
      </div>
    </section>
  </main>

  <dialog id="search-modal" class="modal-dialog">
    <div class="search-header">
      <span style="font-size:1.2rem">🔍</span>
      <input type="search" id="search-query" placeholder="Search the Catechism..." autocomplete="off">
      <button type="button" id="btn-close-search" class="btn-icon">✕</button>
    </div>
    <div class="search-controls-bar">
      <label class="search-option-label">
        <input type="checkbox" id="search-whole-words" checked>
        <span>Whole words</span>
      </label>
      <span id="search-count" class="search-count"></span>
    </div>
    <div id="search-results" class="search-results"></div>
  </dialog>

  <script src="assets/reader.js"></script>
  <script src="assets/search.js"></script>
</body>
</html>`, tocListHTML.String())
}

func parseTOC(doc *nethtml.Node) []*TOCNode {
	var roots []*TOCNode
	var walk func(n *nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "toc-branch") {
					isPart := false
					for _, a := range n.Attr {
						if a.Key == "data-part" {
							isPart = true
						}
					}
					if isPart {
						roots = append(roots, parseBranchNode(n))
						return
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return roots
}

func parseBranchNode(n *nethtml.Node) *TOCNode {
	node := &TOCNode{IsBranch: true, SecNum: -1}
	for _, a := range n.Attr {
		if a.Key == "data-part" {
			node.IsPart = true
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode && c.Data == "div" && hasClass(c, "toc-row") {
			extractRowData(c, node)
		}
		if c.Type == nethtml.ElementNode && c.Data == "div" && hasClass(c, "toc-children") {
			for sub := c.FirstChild; sub != nil; sub = sub.NextSibling {
				if sub.Type == nethtml.ElementNode && sub.Data == "div" {
					if hasClass(sub, "toc-branch") {
						node.Children = append(node.Children, parseBranchNode(sub))
					} else if hasClass(sub, "toc-item") {
						node.Children = append(node.Children, parseItemNode(sub))
					}
				}
			}
		}
	}
	return node
}

func parseItemNode(n *nethtml.Node) *TOCNode {
	node := &TOCNode{IsBranch: false, SecNum: -1}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode && c.Data == "div" && hasClass(c, "toc-row") {
			extractRowData(c, node)
		}
	}
	return node
}

func extractRowData(row *nethtml.Node, node *TOCNode) {
	var findA func(n *nethtml.Node)
	findA = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					node.ID = strings.TrimPrefix(a.Val, "#")
					base := node.ID
					if idx := strings.Index(base, "-heading-"); idx != -1 {
						base = base[:idx]
					}
					if strings.HasPrefix(base, "s-") {
						node.SecNum, _ = strconv.Atoi(strings.TrimPrefix(base, "s-"))
					}
				}
			}
			for sub := n.FirstChild; sub != nil; sub = sub.NextSibling {
				if sub.Type == nethtml.ElementNode && sub.Data == "span" {
					for _, attr := range sub.Attr {
						if attr.Key == "data-toc-title" {
							node.Title = getTextContent(sub)
						}
						if attr.Key == "class" && attr.Val == "toc-range" {
							node.Range = getTextContent(sub)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findA(c)
		}
	}
	findA(row)
}

func hasClass(n *nethtml.Node, cls string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, f := range strings.Fields(a.Val) {
				if f == cls {
					return true
				}
			}
		}
	}
	return false
}

func getTextContent(n *nethtml.Node) string {
	var sb strings.Builder
	var walk func(*nethtml.Node)
	walk = func(curr *nethtml.Node) {
		if curr.Type == nethtml.TextNode {
			sb.WriteString(curr.Data)
		}
		for c := curr.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(html.UnescapeString(sb.String()))
}

func cleanHTMLText(s string) string {
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	cleaned := tagRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(cleaned))
}
