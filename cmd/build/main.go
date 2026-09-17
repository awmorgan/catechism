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

type CanonicalMeta struct {
	Part        string
	SectionName string
	ChapterName string
	ArticleName string
	Title       string
	StartSecID  string
}

type PageDef struct {
	Index       int
	Filename    string
	Part        string
	SectionName string
	ChapterName string
	ArticleName string
	Title       string
	StartID     string
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
	// 001: Prologue
	{Part: "Prologue", SectionName: "", ChapterName: "", ArticleName: "", Title: "Prologue", StartSecID: "s-0"},

	// Part One · Section One
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter One · Man's Capacity for God", ArticleName: "", Title: "Chapter One: Man's Capacity for God", StartSecID: "s-9"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Two · God Comes to Meet Man", ArticleName: "Article 1 · The Revelation of God", Title: "Article 1: The Revelation of God", StartSecID: "s-16"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Two · God Comes to Meet Man", ArticleName: "Article 2 · The Transmission of Divine Revelation", Title: "Article 2: The Transmission of Divine Revelation", StartSecID: "s-21"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Two · God Comes to Meet Man", ArticleName: "Article 3 · Sacred Scripture", Title: "Article 3: Sacred Scripture", StartSecID: "s-27"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Three · Man's Response to God", ArticleName: "Article 1 · I Believe", Title: "Article 1: I Believe", StartSecID: "s-35"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section One · \"I Believe\" — \"We Believe\"", ChapterName: "Chapter Three · Man's Response to God", ArticleName: "Article 2 · We Believe", Title: "Article 2: We Believe", StartSecID: "s-39"},

	// Part One · Section Two · Chapter One: I Believe in God the Father
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", ArticleName: "Article 1 · I Believe in God the Father Almighty, Creator of Heaven and Earth", Title: "Paragraph 1: I Believe in God", StartSecID: "s-46"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", ArticleName: "Article 1 · I Believe in God the Father Almighty, Creator of Heaven and Earth", Title: "Paragraph 2: The Father", StartSecID: "s-50"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", ArticleName: "Article 1 · I Believe in God the Father Almighty, Creator of Heaven and Earth", Title: "Paragraph 3: The Almighty", StartSecID: "s-51"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", ArticleName: "Article 1 · I Believe in God the Father Almighty, Creator of Heaven and Earth", Title: "Paragraph 4: The Creator", StartSecID: "s-52"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", ArticleName: "Article 1 · I Believe in God the Father Almighty, Creator of Heaven and Earth", Title: "Paragraph 5: Heaven and Earth", StartSecID: "s-53"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", ArticleName: "Article 1 · I Believe in God the Father Almighty, Creator of Heaven and Earth", Title: "Paragraph 6: Man", StartSecID: "s-54"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter One · I Believe in God the Father", ArticleName: "Article 1 · I Believe in God the Father Almighty, Creator of Heaven and Earth", Title: "Paragraph 7: The Fall", StartSecID: "s-55"},

	// Part One · Section Two · Chapter Two: I Believe in Jesus Christ, the Only Son of God
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 2 · \"And in Jesus Christ, His Only Son, Our Lord\"", Title: "Article 2: And in Jesus Christ, His Only Son, Our Lord", StartSecID: "s-56"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 3 · \"He Was Conceived by the Power of the Holy Spirit...\"", Title: "Paragraph 1: The Son of God Became Man", StartSecID: "s-63"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 3 · \"He Was Conceived by the Power of the Holy Spirit...\"", Title: "Paragraph 2: Conceived by the Power of the Holy Spirit and Born of the Virgin Mary", StartSecID: "s-65"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 3 · \"He Was Conceived by the Power of the Holy Spirit...\"", Title: "Paragraph 3: The Mysteries of Christ's Life", StartSecID: "s-66"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 4 · \"Jesus Christ Suffered Under Pontius Pilate...\"", Title: "Paragraph 1: Jesus and Israel", StartSecID: "s-67"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 4 · \"Jesus Christ Suffered Under Pontius Pilate...\"", Title: "Paragraph 2: Jesus Died Crucified", StartSecID: "s-70"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 4 · \"Jesus Christ Suffered Under Pontius Pilate...\"", Title: "Paragraph 3: Jesus Christ Was Buried", StartSecID: "s-71"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 5 · \"He Descended into Hell; on the Third Day He Rose...\"", Title: "Paragraph 1: Christ Descended into Hell", StartSecID: "s-72"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 5 · \"He Descended into Hell; on the Third Day He Rose...\"", Title: "Paragraph 2: On the Third Day He Rose from the Dead", StartSecID: "s-75"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 6 · \"He Ascended into Heaven...\"", Title: "Article 6: He Ascended into Heaven, Sits at the Right Hand", StartSecID: "s-76"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Two · I Believe in Jesus Christ, the Only Son of God", ArticleName: "Article 7 · \"From Thence He Will Come Again to Judge...\"", Title: "Article 7: From Thence He Will Come to Judge", StartSecID: "s-79"},

	// Part One · Section Two · Chapter Three: I Believe in the Holy Spirit
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 8 · \"I Believe in the Holy Spirit\"", Title: "Article 8: I Believe in the Holy Spirit", StartSecID: "s-83"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 9 · \"I Believe in the Holy Catholic Church\"", Title: "Paragraph 1: The Church in God's Plan", StartSecID: "s-92"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 9 · \"I Believe in the Holy Catholic Church\"", Title: "Paragraph 2: The Church — People of God, Body of Christ", StartSecID: "s-95"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 9 · \"I Believe in the Holy Catholic Church\"", Title: "Paragraph 3: The Church Is One, Holy, Catholic, Apostolic", StartSecID: "s-96"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 9 · \"I Believe in the Holy Catholic Church\"", Title: "Paragraph 4: Christ's Faithful — Hierarchy, Laity, Consecrated", StartSecID: "s-97"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 9 · \"I Believe in the Holy Catholic Church\"", Title: "Paragraph 5: The Communion of Saints", StartSecID: "s-98"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 9 · \"I Believe in the Holy Catholic Church\"", Title: "Paragraph 6: Mary — Mother of Christ, Mother of the Church", StartSecID: "s-99"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 10 · \"I Believe in the Forgiveness of Sins\"", Title: "Article 10: I Believe in the Forgiveness of Sins", StartSecID: "s-100"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 11 · \"I Believe in the Resurrection of the Body\"", Title: "Article 11: I Believe in the Resurrection of the Body", StartSecID: "s-104"},
	{Part: "Part One · The Profession of Faith", SectionName: "Section Two · The Profession of the Christian Faith", ChapterName: "Chapter Three · I Believe in the Holy Spirit", ArticleName: "Article 12 · \"I Believe in Life Everlasting\"", Title: "Article 12: I Believe in Life Everlasting", StartSecID: "s-109"},

	// Part Two · Section One
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter One · The Paschal Mystery in the Age of the Church", ArticleName: "Article 1 · The Liturgy — Work of the Holy Trinity", Title: "Article 1: The Liturgy — Work of the Holy Trinity", StartSecID: "s-119"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter One · The Paschal Mystery in the Age of the Church", ArticleName: "Article 2 · The Paschal Mystery in the Church's Sacraments", Title: "Article 2: The Paschal Mystery in the Church's Sacraments", StartSecID: "s-127"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter Two · The Sacramental Celebration of the Paschal Mystery", ArticleName: "Article 1 · Celebrating the Church's Liturgy", Title: "Article 1: Celebrating the Church's Liturgy", StartSecID: "s-135"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section One · The Sacramental Economy", ChapterName: "Chapter Two · The Sacramental Celebration of the Paschal Mystery", ArticleName: "Article 2 · Liturgical Diversity and the Unity of the Mystery", Title: "Article 2: Liturgical Diversity and the Unity of the Mystery", StartSecID: "s-142"},

	// Part Two · Section Two
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter One · The Sacraments of Christian Initiation", ArticleName: "Article 1 · The Sacrament of Baptism", Title: "Article 1: The Sacrament of Baptism", StartSecID: "s-145"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter One · The Sacraments of Christian Initiation", ArticleName: "Article 2 · The Sacrament of Confirmation", Title: "Article 2: The Sacrament of Confirmation", StartSecID: "s-157"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter One · The Sacraments of Christian Initiation", ArticleName: "Article 3 · The Sacrament of the Eucharist", Title: "Article 3: The Sacrament of the Eucharist", StartSecID: "s-165"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Two · The Sacraments of Healing", ArticleName: "Article 4 · The Sacrament of Penance and Reconciliation", Title: "Article 4: The Sacrament of Penance and Reconciliation", StartSecID: "s-175"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Two · The Sacraments of Healing", ArticleName: "Article 5 · The Anointing of the Sick", Title: "Article 5: The Anointing of the Sick", StartSecID: "s-190"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Three · The Sacraments at the Service of Communion", ArticleName: "Article 6 · The Sacrament of Holy Orders", Title: "Article 6: The Sacrament of Holy Orders", StartSecID: "s-198"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Three · The Sacraments at the Service of Communion", ArticleName: "Article 7 · The Sacrament of Matrimony", Title: "Article 7: The Sacrament of Matrimony", StartSecID: "s-209"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Four · Other Liturgical Celebrations", ArticleName: "Article 1 · Sacramentals", Title: "Article 1: Sacramentals", StartSecID: "s-218"},
	{Part: "Part Two · The Celebration of the Christian Mystery", SectionName: "Section Two · The Seven Sacraments of the Church", ChapterName: "Chapter Four · Other Liturgical Celebrations", ArticleName: "Article 2 · Christian Funerals", Title: "Article 2: Christian Funerals", StartSecID: "s-222"},

	// Part Three · Section One
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 1 · Man, the Image of God", Title: "Article 1: Man, the Image of God", StartSecID: "s-226"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 2 · Our Vocation to Beatitude", Title: "Article 2: Our Vocation to Beatitude", StartSecID: "s-232"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 3 · Man's Freedom", Title: "Article 3: Man's Freedom", StartSecID: "s-237"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 4 · The Morality of Human Acts", Title: "Article 4: The Morality of Human Acts", StartSecID: "s-242"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 5 · The Morality of the Passions", Title: "Article 5: The Morality of the Passions", StartSecID: "s-247"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 6 · Moral Conscience", Title: "Article 6: Moral Conscience", StartSecID: "s-252"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 7 · The Virtues", Title: "Article 7: The Virtues", StartSecID: "s-259"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter One · The Dignity of the Human Person", ArticleName: "Article 8 · Sin", Title: "Article 8: Sin", StartSecID: "s-265"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Two · The Human Community", ArticleName: "Article 1 · The Person and Society", Title: "Article 1: The Person and Society", StartSecID: "s-272"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Two · The Human Community", ArticleName: "Article 2 · Participation in Social Life", Title: "Article 2: Participation in Social Life", StartSecID: "s-277"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Two · The Human Community", ArticleName: "Article 3 · Social Justice", Title: "Article 3: Social Justice", StartSecID: "s-282"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Three · God's Salvation: Law and Grace", ArticleName: "Article 1 · The Moral Law", Title: "Article 1: The Moral Law", StartSecID: "s-288"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Three · God's Salvation: Law and Grace", ArticleName: "Article 2 · Grace and Justification", Title: "Article 2: Grace and Justification", StartSecID: "s-295"},
	{Part: "Part Three · Life in Christ", SectionName: "Section One · Man's Vocation: Life in the Spirit", ChapterName: "Chapter Three · God's Salvation: Law and Grace", ArticleName: "Article 3 · The Church, Mother and Teacher", Title: "Article 3: The Church, Mother and Teacher", StartSecID: "s-301"},

	// Part Three · Section Two: The Ten Commandments
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter One · \"You Shall Love the Lord Your God...\"", ArticleName: "Article 1 · The First Commandment", Title: "Article 1: The First Commandment", StartSecID: "s-307"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter One · \"You Shall Love the Lord Your God...\"", ArticleName: "Article 2 · The Second Commandment", Title: "Article 2: The Second Commandment", StartSecID: "s-317"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter One · \"You Shall Love the Lord Your God...\"", ArticleName: "Article 3 · The Third Commandment", Title: "Article 3: The Third Commandment", StartSecID: "s-323"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", ArticleName: "Article 4 · The Fourth Commandment", Title: "Article 4: The Fourth Commandment", StartSecID: "s-328"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", ArticleName: "Article 5 · The Fifth Commandment", Title: "Article 5: The Fifth Commandment", StartSecID: "s-337"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", ArticleName: "Article 6 · The Sixth Commandment", Title: "Article 6: The Sixth Commandment", StartSecID: "s-343"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", ArticleName: "Article 7 · The Seventh Commandment", Title: "Article 7: The Seventh Commandment", StartSecID: "s-350"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", ArticleName: "Article 8 · The Eighth Commandment", Title: "Article 8: The Eighth Commandment", StartSecID: "s-359"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", ArticleName: "Article 9 · The Ninth Commandment", Title: "Article 9: The Ninth Commandment", StartSecID: "s-368"},
	{Part: "Part Three · Life in Christ", SectionName: "Section Two · The Ten Commandments", ChapterName: "Chapter Two · \"You Shall Love Your Neighbor as Yourself\"", ArticleName: "Article 10 · The Tenth Commandment", Title: "Article 10: The Tenth Commandment", StartSecID: "s-373"},

	// Part Four · Section One
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter One · The Revelation of Prayer", ArticleName: "Article 1 · In the Old Testament", Title: "Article 1: In the Old Testament", StartSecID: "s-380"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter One · The Revelation of Prayer", ArticleName: "Article 2 · In the Fullness of Time", Title: "Article 2: In the Fullness of Time", StartSecID: "s-386"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter One · The Revelation of Prayer", ArticleName: "Article 3 · In the Age of the Church", Title: "Article 3: In the Age of the Church", StartSecID: "s-389"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Two · The Tradition of Prayer", ArticleName: "Article 1 · At the Wellsprings of Prayer", Title: "Article 1: At the Wellsprings of Prayer", StartSecID: "s-397"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Two · The Tradition of Prayer", ArticleName: "Article 2 · The Way of Prayer", Title: "Article 2: The Way of Prayer", StartSecID: "s-401"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Two · The Tradition of Prayer", ArticleName: "Article 3 · Guides for Prayer", Title: "Article 3: Guides for Prayer", StartSecID: "s-404"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Three · The Life of Prayer", ArticleName: "Article 1 · Expressions of Prayer", Title: "Article 1: Expressions of Prayer", StartSecID: "s-407"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Three · The Life of Prayer", ArticleName: "Article 2 · The Battle of Prayer", Title: "Article 2: The Battle of Prayer", StartSecID: "s-413"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section One · Prayer in the Christian Life", ChapterName: "Chapter Three · The Life of Prayer", ArticleName: "Article 3 · The Prayer of the Hour of Jesus", Title: "Article 3: The Prayer of the Hour of Jesus", StartSecID: "s-419"},

	// Part Four · Section Two: The Lord's Prayer
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", ArticleName: "Article 1 · The Summary of the Whole Gospel", Title: "Article 1: The Summary of the Whole Gospel", StartSecID: "s-422"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", ArticleName: "Article 2 · \"Our Father Who Art in Heaven\"", Title: "Article 2: \"Our Father Who Art in Heaven\"", StartSecID: "s-428"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", ArticleName: "Article 3 · The Seven Petitions", Title: "Article 3: The Seven Petitions", StartSecID: "s-433"},
	{Part: "Part Four · Christian Prayer", SectionName: "Section Two · The Lord's Prayer: \"Our Father!\"", ChapterName: "", ArticleName: "Article 4 · The Final Doxology", Title: "Article 4: The Final Doxology", StartSecID: "s-440"},
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
	secIndexByID := make(map[string]int)

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
		secIndexByID[id] = len(sections)
		sections = append(sections, sec)
	}

	// 3. Map canonical pages to sections
	var pages []*PageDef
	for i := 0; i < len(canonicalPages); i++ {
		startSec, ok := secIndexByID[canonicalPages[i].StartSecID]
		if !ok {
			log.Fatalf("StartSecID %s not found for page %d (%s)", canonicalPages[i].StartSecID, i+1, canonicalPages[i].Title)
		}
		endSec := len(sections)
		if i+1 < len(canonicalPages) {
			if nextStart, ok := secIndexByID[canonicalPages[i+1].StartSecID]; ok {
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
			Title:       canonicalPages[i].Title,
			Part:        canonicalPages[i].Part,
			SectionName: canonicalPages[i].SectionName,
			ChapterName: canonicalPages[i].ChapterName,
			ArticleName: canonicalPages[i].ArticleName,
			StartID:     canonicalPages[i].StartSecID,
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

	// 4. Build search index & para-map
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

	// 5. Generate HTML files in pages/
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

	// 6. Generate Landing Page index.html
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
	currentArticle := ""

	for _, p := range pages {
		if p.Part != currentPart && p.Part != "" {
			currentPart = p.Part
			currentSection = ""
			currentChapter = ""
			currentArticle = ""
			sb.WriteString(fmt.Sprintf(`<div class="toc-part-header">%s</div>`, html.EscapeString(currentPart)))
		}
		if p.SectionName != currentSection && p.SectionName != "" {
			currentSection = p.SectionName
			currentChapter = ""
			currentArticle = ""
			sb.WriteString(fmt.Sprintf(`<div class="toc-section-header">%s</div>`, html.EscapeString(currentSection)))
		}
		if p.ChapterName != currentChapter && p.ChapterName != "" {
			currentChapter = p.ChapterName
			currentArticle = ""
			sb.WriteString(fmt.Sprintf(`<div class="toc-chapter-header">%s</div>`, html.EscapeString(currentChapter)))
		}
		if p.ArticleName != currentArticle && p.ArticleName != "" {
			currentArticle = p.ArticleName
			// Display an Article banner if this page is a sub-paragraph of the article
			if !strings.EqualFold(strings.TrimSpace(p.Title), strings.TrimSpace(p.ArticleName)) &&
				!strings.HasPrefix(p.Title, "Article ") {
				sb.WriteString(fmt.Sprintf(`<div class="toc-article-header">%s</div>`, html.EscapeString(currentArticle)))
			}
		}

		rangeStr := ""
		if p.MinP > 0 && p.MaxP > 0 {
			rangeStr = fmt.Sprintf("¶ %d–%d", p.MinP, p.MaxP)
		}

		indentClass := ""
		if strings.HasPrefix(p.Title, "Paragraph ") {
			indentClass = " toc-indent-sub"
		}

		sb.WriteString(fmt.Sprintf(`
<a href="%s" class="toc-page-link%s" data-page="%s">
  <span class="toc-link-title">%s</span>
  <span class="toc-link-range">%s</span>
</a>`, p.Filename, indentClass, p.Filename, html.EscapeString(p.Title), rangeStr))
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
		bcHTML.WriteString(` <span class="bc-sep">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span class="bc-part">%s</span>`, html.EscapeString(page.Part)))
	}
	if page.SectionName != "" {
		bcHTML.WriteString(` <span class="bc-sep">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span class="bc-section">%s</span>`, html.EscapeString(page.SectionName)))
	}
	if page.ChapterName != "" {
		bcHTML.WriteString(` <span class="bc-sep">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span class="bc-chapter">%s</span>`, html.EscapeString(page.ChapterName)))
	}
	if page.ArticleName != "" && !strings.EqualFold(page.ArticleName, page.Title) && !strings.HasPrefix(page.Title, "Article ") {
		bcHTML.WriteString(` <span class="bc-sep">›</span> `)
		bcHTML.WriteString(fmt.Sprintf(`<span class="bc-article">%s</span>`, html.EscapeString(page.ArticleName)))
	}

	rangeStr := ""
	if page.MinP > 0 && page.MaxP > 0 {
		rangeStr = fmt.Sprintf("¶ %d–%d", page.MinP, page.MaxP)
	}

	var navHTML strings.Builder
	navHTML.WriteString(`<div class="navigation-footer">`)

	// Current Location Card Centerpiece
	navHTML.WriteString(`<div class="nav-location-card">`)
	navHTML.WriteString(`<div class="nav-location-tag">Current Reading Location</div>`)
	navHTML.WriteString(fmt.Sprintf(`<div class="nav-location-breadcrumbs">%s</div>`, bcHTML.String()))
	navHTML.WriteString(fmt.Sprintf(`<div class="nav-location-title">%s</div>`, html.EscapeString(page.Title)))
	if rangeStr != "" {
		navHTML.WriteString(fmt.Sprintf(`<div class="nav-location-range">%s</div>`, rangeStr))
	}
	navHTML.WriteString(`<button type="button" class="btn-location-toc" id="btn-bottom-toc">📖 Table of Contents</button>`)
	navHTML.WriteString(`</div>`)

	// Prev / Next Navigation Cards
	navHTML.WriteString(`<nav class="page-navigation" aria-label="Chapter Navigation">`)
	if page.Prev != nil {
		prevRange := ""
		if page.Prev.MinP > 0 {
			prevRange = fmt.Sprintf("¶ %d–%d", page.Prev.MinP, page.Prev.MaxP)
		}
		prevContext := page.Prev.ChapterName
		if prevContext == "" {
			prevContext = page.Prev.SectionName
		}
		if page.Prev.ArticleName != "" && prevContext != "" {
			prevContext = page.Prev.ArticleName
		}
		navHTML.WriteString(fmt.Sprintf(`
  <a href="%s" class="nav-card prev" rel="prev">
    <span class="nav-label">← Previous</span>
    <span class="nav-context">%s</span>
    <span class="nav-title">%s</span>
    <span class="nav-range">%s</span>
  </a>`, page.Prev.Filename, html.EscapeString(prevContext), html.EscapeString(page.Prev.Title), prevRange))
	} else {
		navHTML.WriteString(`<div></div>`)
	}

	if page.Next != nil {
		nextRange := ""
		if page.Next.MinP > 0 {
			nextRange = fmt.Sprintf("¶ %d–%d", page.Next.MinP, page.Next.MaxP)
		}
		nextContext := page.Next.ChapterName
		if nextContext == "" {
			nextContext = page.Next.SectionName
		}
		if page.Next.ArticleName != "" && nextContext != "" {
			nextContext = page.Next.ArticleName
		}
		navHTML.WriteString(fmt.Sprintf(`
  <a href="%s" class="nav-card next" rel="next">
    <span class="nav-label">Next →</span>
    <span class="nav-context">%s</span>
    <span class="nav-title">%s</span>
    <span class="nav-range">%s</span>
  </a>`, page.Next.Filename, html.EscapeString(nextContext), html.EscapeString(page.Next.Title), nextRange))
	} else {
		navHTML.WriteString(`<div></div>`)
	}
	navHTML.WriteString(`</nav></div>`)

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

	badge := page.Part
	if page.ArticleName != "" && !strings.EqualFold(page.ArticleName, page.Title) && !strings.HasPrefix(page.Title, "Article ") {
		badge = page.ArticleName
	} else if page.ChapterName != "" {
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
      <span id="active-reading-para" class="active-para-tag" style="display:none;" aria-live="polite"></span>
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
	currentArticle := ""

	for _, p := range pages {
		if p.Part != currentPart && p.Part != "" {
			currentPart = p.Part
			currentSection = ""
			currentChapter = ""
			currentArticle = ""
			tocListHTML.WriteString(fmt.Sprintf(`
<div class="landing-part-header">
  %s
</div>`, html.EscapeString(currentPart)))
		}
		if p.SectionName != currentSection && p.SectionName != "" {
			currentSection = p.SectionName
			currentChapter = ""
			currentArticle = ""
			tocListHTML.WriteString(fmt.Sprintf(`
<div class="landing-section-header">
  %s
</div>`, html.EscapeString(currentSection)))
		}
		if p.ChapterName != currentChapter && p.ChapterName != "" {
			currentChapter = p.ChapterName
			currentArticle = ""
			tocListHTML.WriteString(fmt.Sprintf(`
<div class="landing-chapter-header">
  %s
</div>`, html.EscapeString(currentChapter)))
		}
		if p.ArticleName != currentArticle && p.ArticleName != "" {
			currentArticle = p.ArticleName
			if !strings.EqualFold(strings.TrimSpace(p.Title), strings.TrimSpace(p.ArticleName)) &&
				!strings.HasPrefix(p.Title, "Article ") {
				tocListHTML.WriteString(fmt.Sprintf(`
<div class="landing-article-header">
  %s
</div>`, html.EscapeString(currentArticle)))
			}
		}

		rangeStr := ""
		if p.MinP > 0 && p.MaxP > 0 {
			rangeStr = fmt.Sprintf("¶ %d–%d", p.MinP, p.MaxP)
		}

		indentClass := ""
		if strings.HasPrefix(p.Title, "Paragraph ") {
			indentClass = " toc-indent-sub"
		}

		tocListHTML.WriteString(fmt.Sprintf(`
<a href="pages/%s" class="toc-page-link%s">
  <span class="toc-link-title">%s</span>
  <span class="toc-link-range">%s</span>
</a>`, p.Filename, indentClass, html.EscapeString(p.Title), rangeStr))
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
      <div class="landing-toc-container">
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
