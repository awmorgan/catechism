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

type CanonicalMeta struct {
	Part        string
	SectionName string
	ChapterName string
	Title       string
}

type PageDef struct {
	Index       int
	Filename    string
	Part        string
	SectionName string
	ChapterName string
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

var canonicalPages = []CanonicalMeta{
	// 001
	{Part: "Prologue", SectionName: "", ChapterName: "", Title: "Prologue"},
	// Part One · Section One
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter One · Man's Capacity for God", Title: "Chapter One: Man's Capacity for God"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Two · God Comes to Meet Man", Title: "Article 1: The Revelation of God"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Two · God Comes to Meet Man", Title: "Article 2: The Transmission of Divine Revelation"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Two · God Comes to Meet Man", Title: "Article 3: Sacred Scripture"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Three · Man's Response to God", Title: "Article 1: I Believe"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Three · Man's Response to God", Title: "Article 2: We Believe"},
	// Part One · Section Two · Chapter One
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", Title: "Paragraph 1: I Believe in God"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", Title: "Paragraph 2: The Father"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", Title: "Paragraph 3: The Almighty"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", Title: "Paragraph 4: The Creator"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", Title: "Paragraph 5: Heaven and Earth"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", Title: "Paragraph 6: Man"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", Title: "Paragraph 7: The Fall"},
	// Part One · Section Two · Chapter Two
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Article 2: And in Jesus Christ, His Only Son, Our Lord"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Paragraph 1: Jesus"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Paragraph 2: Christ"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Article 3: Conceived by the Holy Spirit, Born of the Virgin Mary"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Paragraph 3: The Mysteries of Christ's Life"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Article 4: Jesus Suffered Under Pontius Pilate, Was Crucified"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Paragraph 2: Jesus Died Crucified"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Paragraph 3: Jesus Christ Was Buried"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Article 5: Jesus Descended into Hell; Rose on the Third Day"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Paragraph 2: On the Third Day He Rose from the Dead"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Article 6: He Ascended into Heaven, Sits at the Right Hand"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", Title: "Article 7: From Thence He Will Come to Judge"},
	// Part One · Section Two · Chapter Three
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Article 8: I Believe in the Holy Spirit"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Article 9, Paragraph 1: The Church in God's Plan"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Paragraph 2: The Church — People of God, Body of Christ"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Paragraph 3: The Church Is One, Holy, Catholic, Apostolic"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Paragraph 4: Christ's Faithful — Hierarchy, Laity, Consecrated"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Paragraph 5: The Communion of Saints"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Paragraph 6: Mary — Mother of Christ, Mother of the Church"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Article 10: I Believe in the Forgiveness of Sins"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Article 11: I Believe in the Resurrection of the Body"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", Title: "Article 12: I Believe in Life Everlasting"},
	// Part Two · Section One
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter One · The Paschal Mystery in the Age of the Church", Title: "Article 1: The Liturgy — Work of the Holy Trinity"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter One · The Paschal Mystery in the Age of the Church", Title: "Article 2: The Paschal Mystery in the Church's Sacraments"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter Two · The Sacramental Celebration of the Paschal Mystery", Title: "Article 1: Celebrating the Church's Liturgy"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter Two · The Sacramental Celebration of the Paschal Mystery", Title: "Article 2: Liturgical Diversity and the Unity of the Mystery"},
	// Part Two · Section Two
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter One · The Sacraments of Christian Initiation", Title: "Article 1: The Sacrament of Baptism"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter One · The Sacraments of Christian Initiation", Title: "Article 2: The Sacrament of Confirmation"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter One · The Sacraments of Christian Initiation", Title: "Article 3: The Sacrament of the Eucharist"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Two · The Sacraments of Healing", Title: "Article 4: The Sacrament of Penance and Reconciliation"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Two · The Sacraments of Healing", Title: "Article 5: The Anointing of the Sick"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Three · The Sacraments at the Service of Communion", Title: "Article 6: The Sacrament of Holy Orders"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Three · The Sacraments at the Service of Communion", Title: "Article 7: The Sacrament of Matrimony"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Four · Other Liturgical Celebrations", Title: "Article 1: Sacramentals"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Four · Other Liturgical Celebrations", Title: "Article 2: Christian Funerals"},
	// Part Three · Section One
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 1: Man, the Image of God"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 2: Our Vocation to Beatitude"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 3: Man's Freedom"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 4: The Morality of Human Acts"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 5: The Morality of the Passions"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 6: Moral Conscience"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 7: The Virtues"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", Title: "Article 8: Sin"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Two · The Human Community", Title: "Article 1: The Person and Society"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Two · The Human Community", Title: "Article 2: Participation in Social Life"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Two · The Human Community", Title: "Article 3: Social Justice"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Three · God's Salvation: Law and Grace", Title: "Article 1: The Moral Law"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Three · God's Salvation: Law and Grace", Title: "Article 2: Grace and Justification"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Three · God's Salvation: Law and Grace", Title: "Article 3: The Church, Mother and Teacher"},
	// Part Three · Section Two: The Ten Commandments
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter One · \"You Shall Love the Lord Your God...\"", Title: "Article 1: The First Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter One · \"You Shall Love the Lord Your God...\"", Title: "Article 2: The Second Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter One · \"You Shall Love the Lord Your God...\"", Title: "Article 3: The Third Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", Title: "Article 4: The Fourth Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", Title: "Article 5: The Fifth Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", Title: "Article 6: The Sixth Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", Title: "Article 7: The Seventh Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", Title: "Article 8: The Eighth Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", Title: "Article 9: The Ninth Commandment"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", Title: "Article 10: The Tenth Commandment"},
	// Part Four · Section One
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter One · The Revelation of Prayer", Title: "Article 1: In the Old Testament"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter One · The Revelation of Prayer", Title: "Article 2: In the Fullness of Time"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter One · The Revelation of Prayer", Title: "Article 3: In the Age of the Church"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Two · The Tradition of Prayer", Title: "Article 1: At the Wellsprings of Prayer"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Two · The Tradition of Prayer", Title: "Article 2: The Way of Prayer"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Two · The Tradition of Prayer", Title: "Article 3: Guides for Prayer"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Three · The Life of Prayer", Title: "Article 1: Expressions of Prayer"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Three · The Life of Prayer", Title: "Article 2: The Battle of Prayer"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Three · The Life of Prayer", Title: "Article 3: The Prayer of the Hour of Jesus"},
	// Part Four · Section Two: The Lord's Prayer
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", Title: "Article 1: The Summary of the Whole Gospel"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", Title: "Article 2: \"Our Father Who Art in Heaven\""},
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", Title: "Article 3: The Seven Petitions"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", Title: "Article 4: The Final Doxology"},
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

	// Slicing sections into contiguous pages
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
		part := ""
		secName := ""
		chapName := ""
		title := rawPages[i].Title
		if i < len(canonicalPages) {
			part = canonicalPages[i].Part
			secName = canonicalPages[i].SectionName
			chapName = canonicalPages[i].ChapterName
			title = canonicalPages[i].Title
		}

		page := &PageDef{
			Index:       i + 1,
			Filename:    filename,
			Title:       title,
			Part:        part,
			SectionName: secName,
			ChapterName: chapName,
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
	currentSection := ""
	currentChapter := ""

	for _, p := range pages {
		if p.Part != currentPart && p.Part != "" {
			currentPart = p.Part
			currentSection = ""
			currentChapter = ""
			sb.WriteString(fmt.Sprintf(`<div class="toc-part-header">%s</div>`, html.EscapeString(currentPart)))
		}
		if p.SectionName != currentSection && p.SectionName != "" {
			currentSection = p.SectionName
			currentChapter = ""
			sb.WriteString(fmt.Sprintf(`<div class="toc-section-header">%s</div>`, html.EscapeString(currentSection)))
		}
		if p.ChapterName != currentChapter && p.ChapterName != "" {
			currentChapter = p.ChapterName
			sb.WriteString(fmt.Sprintf(`<div class="toc-chapter-header">%s</div>`, html.EscapeString(currentChapter)))
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
	if page.Part != "" && page.Part != "Prologue" {
		bcHTML.WriteString(` <span style="opacity:0.5">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span>%s</span>`, html.EscapeString(page.Part)))
	}
	if page.SectionName != "" {
		bcHTML.WriteString(` <span style="opacity:0.5">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span>%s</span>`, html.EscapeString(page.SectionName)))
	}
	if page.ChapterName != "" {
		bcHTML.WriteString(` <span style="opacity:0.5">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span>%s</span>`, html.EscapeString(page.ChapterName)))
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
	} else {
		navHTML.WriteString(`<div></div>`)
	}
	navHTML.WriteString(`</nav>`)

	var bodyContent strings.Builder
	leadingH2Regex := regexp.MustCompile(`(?s)^\s*<h2[^>]*>.*?</h2>\s*`)
	preamblePRegex := regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)

	for secIdx, s := range page.Sections {
		bodyContent.WriteString(fmt.Sprintf(`<section class="chapter" id="%s">`, s.ID))
		secHTML := s.HTML
		if secIdx == 0 {
			// Strip leading redundant <h2> that repeats the page <h1> title
			secHTML = leadingH2Regex.ReplaceAllString(secHTML, "")
		}

		// Deduplicate redundant preamble paragraphs before the first numbered paragraph
		divIdx := strings.Index(secHTML, `<div class="text">`)
		if divIdx != -1 {
			prefix := secHTML[:divIdx+len(`<div class="text">`)]
			remainder := secHTML[divIdx+len(`<div class="text">`):]

			firstP := strings.Index(remainder, `<p id="p-`)
			if firstP == -1 {
				firstP = strings.Index(remainder, `<details`)
			}
			if firstP == -1 {
				firstP = len(remainder)
			}

			preamble := remainder[:firstP]
			postPreamble := remainder[firstP:]

			normH2 := normText(s.Heading)
			normPage := normText(page.Title)

			cleanedPreamble := preamblePRegex.ReplaceAllStringFunc(preamble, func(pTag string) string {
				m := preamblePRegex.FindStringSubmatch(pTag)
				if len(m) < 2 {
					return pTag
				}
				inner := m[1]
				normP := normText(inner)

				// Strip empty or stray punctuation
				if normP == "" || normP == "?" {
					return ""
				}
				// Strip exact match with section H2 or page H1
				if (normH2 != "" && normH2 == normP) || (normPage != "" && normPage == normP) {
					return ""
				}
				// Strip bold heading banners matching H2 or Page Title
				if (strings.Contains(inner, "<b>") || strings.Contains(inner, "<strong>")) &&
					((normH2 != "" && (strings.Contains(normH2, normP) || strings.Contains(normP, normH2))) ||
						(normPage != "" && (strings.Contains(normPage, normP) || strings.Contains(normP, normPage)))) &&
					(strings.HasPrefix(normP, "article") || strings.HasPrefix(normP, "section") || strings.HasPrefix(normP, "chapter") || strings.HasPrefix(normP, "part") || len(normP) > 10) {
					return ""
				}
				return pTag
			})

			secHTML = prefix + cleanedPreamble + postPreamble
		}

		bodyContent.WriteString(secHTML)
		bodyContent.WriteString(`</section>`)
	}

	rangeStr := ""
	if page.MinP > 0 && page.MaxP > 0 {
		rangeStr = fmt.Sprintf("¶ %d–%d", page.MinP, page.MaxP)
	}

	badge := page.Part
	if page.ChapterName != "" {
		badge = page.ChapterName
	} else if page.SectionName != "" {
		badge = page.SectionName
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
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
        <form id="jump-form" class="jump-form" action="javascript:void(0);">
          <input type="text" inputmode="numeric" pattern="[0-9]*" id="jump-input" class="jump-input" placeholder="¶ 1–2865" aria-label="Jump to paragraph number" enterkeyhint="go">
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
		html.EscapeString(badge),
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
	currentSection := ""
	currentChapter := ""
	for _, p := range pages {
		if p.Part != currentPart && p.Part != "" {
			currentPart = p.Part
			currentSection = ""
			currentChapter = ""
			tocListHTML.WriteString(fmt.Sprintf(`
<div class="landing-part-header" style="margin: 2.2rem 0 0.6rem; padding-bottom: 0.4rem; border-bottom: 2px solid var(--accent); font-family: var(--font-sans); font-size: 1.25rem; font-weight: 700; color: var(--accent); text-transform: uppercase; letter-spacing: 0.03em;">
  %s
</div>`, html.EscapeString(currentPart)))
		}
		if p.SectionName != currentSection && p.SectionName != "" {
			currentSection = p.SectionName
			currentChapter = ""
			tocListHTML.WriteString(fmt.Sprintf(`
<div class="landing-section-header" style="margin: 1.3rem 0 0.45rem 0.25rem; font-family: var(--font-sans); font-size: 1.05rem; font-weight: 700; color: var(--text); border-bottom: 1px dashed var(--border); padding-bottom: 0.25rem;">
  %s
</div>`, html.EscapeString(currentSection)))
		}
		if p.ChapterName != currentChapter && p.ChapterName != "" {
			currentChapter = p.ChapterName
			tocListHTML.WriteString(fmt.Sprintf(`
<div class="landing-chapter-header" style="margin: 0.85rem 0 0.35rem 0.5rem; font-family: var(--font-sans); font-size: 0.9rem; font-weight: 600; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.04em;">
  %s
</div>`, html.EscapeString(currentChapter)))
		}

		rangeStr := ""
		if p.MinP > 0 && p.MaxP > 0 {
			rangeStr = fmt.Sprintf("¶ %d–%d", p.MinP, p.MaxP)
		}

		tocListHTML.WriteString(fmt.Sprintf(`
<a href="pages/%s" class="toc-page-link" style="background: var(--bg-card); border: 1px solid var(--border); padding: 0.75rem 1rem; margin-bottom: 0.35rem; margin-left: 0.5rem; border-radius: 8px;">
  <span class="toc-link-title" style="font-size: 0.95rem; font-weight: 500;">%s</span>
  <span class="toc-link-range" style="font-size: 0.82rem; background: var(--accent-light); color: var(--accent); padding: 0.15rem 0.45rem; border-radius: 4px;">%s</span>
</a>`, p.Filename, html.EscapeString(p.Title), rangeStr))
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
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
        <form id="jump-form" class="jump-form" action="javascript:void(0);">
          <input type="text" inputmode="numeric" pattern="[0-9]*" id="jump-input" class="jump-input" placeholder="Go to ¶ (1–2865)" style="height: 48px; width: 8.5rem; font-size: 1rem;" enterkeyhint="go">
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

func normText(s string) string {
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	cleaned := strings.ToLower(tagRegex.ReplaceAllString(s, ""))
	var sb strings.Builder
	for _, r := range cleaned {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
