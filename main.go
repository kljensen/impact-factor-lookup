package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type JournalMetrics struct {
	Title        string   `db:"title"`
	Field        int64    `db:"field"`
	Year         int64    `db:"year"`
	SJR          float64  `db:"sjr"`
	HIndex       int64    `db:"h_index"`
	AvgCitations float64  `db:"avg_citations"`
	ISSNs        []string `db:"issn"` // Splitting the comma-separated ISSNs into a slice
	SourceID     int64    `db:"sourceid"`
}

// Helper function to parse comma-separated ISSNs into a slice
func parseISSNs(issnString string) []string {
	// Remove any whitespace and split by comma
	issns := strings.Split(strings.ReplaceAll(issnString, " ", ""), ",")
	// Clean up any empty strings
	var result []string
	for _, issn := range issns {
		if issn != "" {
			result = append(result, issn)
		}
	}
	return result
}

// Function to create a new JournalMetrics from raw data
func NewJournalMetrics(title string, field, year int64, sjr float64, hIndex int64,
	avgCitations float64, issnString string, sourceID int64) JournalMetrics {

	return JournalMetrics{
		Title:        title,
		Field:        field,
		Year:         year,
		SJR:          sjr,
		HIndex:       hIndex,
		AvgCitations: avgCitations,
		ISSNs:        parseISSNs(issnString),
		SourceID:     sourceID,
	}
}

// Add a map type for easy ISSN lookup
type MetricsDatabase map[string]JournalMetrics

// Add a lookup function to the database
func (db MetricsDatabase) LookupISSN(issn string) (JournalMetrics, bool) {
	// remove non-numeric characters from the ISSN
	issn = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, issn)
	// keys in the database are the cleaned-up ISSNs
	jm, ok := db[issn]
	return jm, ok
}

// Load
func ReadMetricsCSV(filename string) (MetricsDatabase, error) {
	// Open the CSV file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read the header
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	// Create the database
	db := make(MetricsDatabase)

	// Read the rest of the records
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading record: %v", err)
		}

		field, err := strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing field value: %v", err)
		}

		year, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing year value: %v", err)
		}

		// Parse the values
		// Assuming the CSV columns are in order:
		// Title,field,year,SJR,h-index,avg_citations,Issn,Sourceid
		sjr := -1.0
		if record[3] != "" {
			sjr, err = strconv.ParseFloat(record[3], 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing SJR value: %v", err)
			}
		}

		hIndex, err := strconv.ParseInt(record[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing h-index value: %v", err)
		}

		avgCitations := -1.0
		if record[5] != "" {
			avgCitations, err = strconv.ParseFloat(record[5], 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing average citations value: %v", err)
			}
		}

		sourceID, err := strconv.ParseInt(record[7], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing sourceID value: %v", err)
		}

		// Create the journal metrics
		// Using 0 for field, year, and sourceID as they're not in the CSV
		metrics := NewJournalMetrics(
			record[0], // Title
			field,
			year,
			sjr,          // SJR
			hIndex,       // h-index
			avgCitations, // avg_citations
			record[6],    // ISSN string
			sourceID,     // SourceID
		)

		// Add each ISSN as a key pointing to this journal's metrics
		for _, issn := range metrics.ISSNs {
			// See if the ISSN is already in the database
			if found, ok := db[issn]; ok {
				if found.Year < metrics.Year {
					db[issn] = metrics
				}
			} else {
				db[issn] = metrics
			}
		}
	}

	return db, nil
}

type OAIPMH struct {
	XMLName      xml.Name    `xml:"OAI-PMH"`
	ResponseDate string      `xml:"responseDate"`
	Request      Request     `xml:"request"`
	ListRecords  ListRecords `xml:"ListRecords"`
}

type Request struct {
	MetadataPrefix string `xml:"metadataPrefix,attr"`
	Verb           string `xml:"verb,attr"`
	Set            string `xml:"set,attr"`
}

type ListRecords struct {
	Records []Record `xml:"record"`
}

type Record struct {
	Header   Header   `xml:"header"`
	Metadata Metadata `xml:"metadata"`
}

type Header struct {
	Identifier string `xml:"identifier"`
	Datestamp  string `xml:"datestamp"`
	SetSpec    string `xml:"setSpec"`
}

type Metadata struct {
	Publication Publication `xml:"Publication"`
}

type Publication struct {
	ID        string      `xml:"id,attr"`
	Type      string      `xml:"Type"`
	Language  string      `xml:"Language"`
	Title     string      `xml:"Title"`
	Subtitle  string      `xml:"Subtitle"`
	Published PublishedIn `xml:"PublishedIn"`
	Date      string      `xml:"PublicationDate"`
	Volume    string      `xml:"Volume"`
	Issue     string      `xml:"Issue"`
	DOI       string      `xml:"DOI"`
	ISSN      string      `xml:"ISSN"`
	URL       string      `xml:"URL"`
	Authors   Authors     `xml:"Authors"`
}

type Authors struct {
	AuthorList []Author `xml:"Author"`
}

type Author struct {
	Person Person `xml:"Person"`
}

type Person struct {
	PersonName PersonName `xml:"PersonName"`
}

type PersonName struct {
	FamilyNames string `xml:"FamilyNames"`
	FirstNames  string `xml:"FirstNames"`
}

type PublishedIn struct {
	Publication JournalInfo `xml:"Publication"`
}

type JournalInfo struct {
	Type  string `xml:"Type"`
	Title string `xml:"Title"`
}

// Function to create a BibTeX citation key
func createCitationKey(pub Publication) string {
	// Get first author's last name or "Unknown"
	authorName := "Unknown"
	if len(pub.Authors.AuthorList) > 0 {
		authorName = pub.Authors.AuthorList[0].Person.PersonName.FamilyNames
	}

	// Get year from date
	year := "0000"
	if len(pub.Date) >= 4 {
		year = pub.Date[0:4]
	}

	// Create base key
	key := fmt.Sprintf("%s%s", authorName, year)

	// Remove spaces and special characters
	key = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, key)

	return key
}

// Function to format authors for BibTeX
func formatAuthors(authors []Author) string {
	var names []string
	for _, author := range authors {
		name := fmt.Sprintf("%s, %s",
			author.Person.PersonName.FamilyNames,
			author.Person.PersonName.FirstNames)
		names = append(names, name)
	}
	return strings.Join(names, " and ")
}

// Function to convert a publication to BibTeX format
func toBibTeX(pub Publication, metrics JournalMetrics) string {
	var bibtex strings.Builder

	// Start entry
	citationKey := createCitationKey(pub)
	bibtex.WriteString(fmt.Sprintf("@article{%s,\n", citationKey))

	// Authors
	if len(pub.Authors.AuthorList) > 0 {
		authors := formatAuthors(pub.Authors.AuthorList)
		bibtex.WriteString(fmt.Sprintf("  author = {%s},\n", authors))
	}

	// Title
	if pub.Title != "" {
		bibtex.WriteString(fmt.Sprintf("  title = {{%s}},\n", pub.Title))
	}

	// Journal
	if pub.Published.Publication.Title != "" {
		bibtex.WriteString(fmt.Sprintf("  journal = {%s},\n", pub.Published.Publication.Title))
	}

	// Year and Month
	if pub.Date != "" {
		// Try to parse the date
		t, err := time.Parse("2006-01-02", pub.Date)
		if err != nil {
			// Try just year-month
			t, err = time.Parse("2006-01", pub.Date)
		}
		if err == nil {
			bibtex.WriteString(fmt.Sprintf("  year = {%d},\n", t.Year()))
			bibtex.WriteString(fmt.Sprintf("  month = {%s},\n", strings.ToLower(t.Month().String())))
		} else {
			// Just use the year part if we have it
			if len(pub.Date) >= 4 {
				bibtex.WriteString(fmt.Sprintf("  year = {%s},\n", pub.Date[0:4]))
			}
		}
	}

	// Volume
	if pub.Volume != "" {
		bibtex.WriteString(fmt.Sprintf("  volume = {%s},\n", pub.Volume))
	}

	// Issue/Number
	if pub.Issue != "" {
		bibtex.WriteString(fmt.Sprintf("  number = {%s},\n", pub.Issue))
	}

	// DOI
	if pub.DOI != "" {
		bibtex.WriteString(fmt.Sprintf("  doi = {%s},\n", pub.DOI))
	}

	// ISSN
	if pub.ISSN != "" {
		bibtex.WriteString(fmt.Sprintf("  issn = {%s},\n", pub.ISSN))
	}

	// Add the impact factor stuff
	bibtex.WriteString(fmt.Sprintf("  sjr = {%f},\n", metrics.SJR))
	bibtex.WriteString(fmt.Sprintf("  avg_citations = {%f},\n", metrics.AvgCitations))
	bibtex.WriteString(fmt.Sprintf("  h_index = {%d},\n", metrics.HIndex))

	// Remove trailing comma and add closing brace
	output := bibtex.String()
	output = strings.TrimSuffix(output, ",\n") + "\n}\n"

	return output
}

// Sort papers by average citations. Takes a slice of publications and a map of journal metrics.
// Returns a slice of publications sorted by average citations.
// If a publication's journal is not found in the metrics map, it is placed at the end.
func sortPapersByCitations(papers []Publication, metrics MetricsDatabase) []Publication {
	// Create a slice of papers with metrics
	var papersWithMetrics []struct {
		pub     Publication
		metrics JournalMetrics
	}
	for _, paper := range papers {
		metrics, ok := metrics.LookupISSN(paper.ISSN)
		if !ok {
			metrics = JournalMetrics{}
		}
		papersWithMetrics = append(papersWithMetrics, struct {
			pub     Publication
			metrics JournalMetrics
		}{pub: paper, metrics: metrics})
	}

	// Sort the papers by average citations
	sort.Slice(papersWithMetrics, func(i, j int) bool {
		return papersWithMetrics[i].metrics.AvgCitations > papersWithMetrics[j].metrics.AvgCitations
	})

	// Extract the sorted papers
	var sortedPapers []Publication
	for _, paper := range papersWithMetrics {
		sortedPapers = append(sortedPapers, paper.pub)
	}

	return sortedPapers
}

func main() {
	// Get file name from os.Args
	if len(os.Args) != 3 {
		log.Printf("Usage: %s <paper xml filename> <impact factor csv>", os.Args[0])
		os.Exit(1)
	}
	xmlFilename := os.Args[1]
	csvFilename := os.Args[2]

	// Read the XML file
	xmlData, err := os.ReadFile(xmlFilename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	journalDB, err := ReadMetricsCSV(csvFilename)
	if err != nil {
		log.Fatalln(err)
	}

	// Parse the XML
	var oaiData OAIPMH
	err = xml.Unmarshal(xmlData, &oaiData)
	if err != nil {
		fmt.Printf("Error parsing XML: %v\n", err)
		return
	}

	// Extract the Publication from each Record
	pubs := make([]Publication, 0, len(oaiData.ListRecords.Records))
	for _, record := range oaiData.ListRecords.Records {
		pubs = append(pubs, record.Metadata.Publication)
	}

	pubs = sortPapersByCitations(pubs, journalDB)

	// Print DOI and ISSN for each paper
	for _, pub := range pubs {
		issn := pub.ISSN
		metrics, _ := journalDB.LookupISSN(issn)
		fmt.Println(toBibTeX(pub, metrics))
	}
}
