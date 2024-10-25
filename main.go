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

// JournalMetrics holds information about the metrics of a journal.
type JournalMetrics struct {
	Title        string   `db:"title"`
	Field        int64    `db:"field"`
	Year         int64    `db:"year"`
	SJR          float64  `db:"sjr"`
	HIndex       int64    `db:"h_index"`
	AvgCitations float64  `db:"avg_citations"`
	ISSNs        []string `db:"issn"` // Split the comma-separated ISSNs into a slice for easy lookup.
	SourceID     int64    `db:"sourceid"`
}

// parseISSNs splits and cleans up a comma-separated ISSN string into a slice.
func parseISSNs(issnString string) []string {
	// Remove any whitespace and split by commas
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

// NewJournalMetrics creates and initializes a new JournalMetrics instance from provided data.
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

// MetricsDatabase is a map-based database for storing journal metrics with ISSNs as keys.
type MetricsDatabase map[string]JournalMetrics

// LookupISSN searches the database for journal metrics by ISSN.
func (db MetricsDatabase) LookupISSN(issn string) (JournalMetrics, bool) {
	// Clean the ISSN by removing non-numeric characters.
	issn = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, issn)
	// Return the corresponding journal metrics if available.
	jm, ok := db[issn]
	return jm, ok
}

// ReadMetricsCSV loads journal metrics from a CSV file into the MetricsDatabase.
func ReadMetricsCSV(filename string) (MetricsDatabase, error) {
	// Open the CSV file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read the header (skipping it as we assume the structure is known)
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	// Initialize the database
	db := make(MetricsDatabase)

	// Read and parse each record from the CSV
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading record: %v", err)
		}

		// Parse each field based on its type
		field, err := strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing field value: %v", err)
		}

		year, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing year value: %v", err)
		}

		// Optional parsing for SJR, defaulting to -1.0 if not present
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

		// Create and populate a JournalMetrics object
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

		// Add each ISSN from the metrics as a key in the database
		for _, issn := range metrics.ISSNs {
			// Check if ISSN already exists, update only if the year is newer
			if existing, ok := db[issn]; ok {
				if existing.Year < metrics.Year {
					db[issn] = metrics
				}
			} else {
				db[issn] = metrics
			}
		}
	}

	return db, nil
}

// Define XML structures based on OAI-PMH response
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

// Publication and nested structures represent the XML data schema
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

// Authors represents a list of authors in a publication.
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

// createCitationKey generates a BibTeX citation key based on the first author's last name and publication year.
func createCitationKey(pub Publication) string {
	// Get first author's last name or "Unknown" if not available
	authorName := "Unknown"
	if len(pub.Authors.AuthorList) > 0 {
		authorName = pub.Authors.AuthorList[0].Person.PersonName.FamilyNames
	}

	// Extract year from the publication date
	year := "0000"
	if len(pub.Date) >= 4 {
		year = pub.Date[0:4]
	}

	// Create and clean the citation key
	key := fmt.Sprintf("%s%s", authorName, year)
	key = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, key)

	return key
}

// formatAuthors formats a list of authors for a BibTeX entry.
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

// toBibTeX converts a publication and its associated metrics to a BibTeX formatted string.
func toBibTeX(pub Publication, metrics JournalMetrics) string {
	var bibtex strings.Builder

	// Begin the BibTeX entry with the citation key
	citationKey := createCitationKey(pub)
	bibtex.WriteString(fmt.Sprintf("@article{%s,\n", citationKey))

	// Add the authors if available
	if len(pub.Authors.AuthorList) > 0 {
		authors := formatAuthors(pub.Authors.AuthorList)
		bibtex.WriteString(fmt.Sprintf("  author = {%s},\n", authors))
	}

	// Add the title
	if pub.Title != "" {
		bibtex.WriteString(fmt.Sprintf("  title = {{%s}},\n", pub.Title))
	}

	// Add the journal information if available
	if pub.Published.Publication.Title != "" {
		bibtex.WriteString(fmt.Sprintf("  journal = {%s},\n", pub.Published.Publication.Title))
	}

	// Parse and format the publication date
	if pub.Date != "" {
		t, err := time.Parse("2006-01-02", pub.Date)
		if err != nil {
			// Attempt to parse as "year-month" if full date fails
			t, err = time.Parse("2006-01", pub.Date)
		}
		if err == nil {
			bibtex.WriteString(fmt.Sprintf("  year = {%d},\n", t.Year()))
			bibtex.WriteString(fmt.Sprintf("  month = {%s},\n", strings.ToLower(t.Month().String())))
		} else if len(pub.Date) >= 4 {
			// Fallback to using just the year if parsing fails
			bibtex.WriteString(fmt.Sprintf("  year = {%s},\n", pub.Date[0:4]))
		}
	}

	// Add volume and issue information if available
	if pub.Volume != "" {
		bibtex.WriteString(fmt.Sprintf("  volume = {%s},\n", pub.Volume))
	}
	if pub.Issue != "" {
		bibtex.WriteString(fmt.Sprintf("  number = {%s},\n", pub.Issue))
	}

	// Include DOI and ISSN if available
	if pub.DOI != "" {
		bibtex.WriteString(fmt.Sprintf("  doi = {%s},\n", pub.DOI))
	}
	if pub.ISSN != "" {
		bibtex.WriteString(fmt.Sprintf("  issn = {%s},\n", pub.ISSN))
	}

	// Add additional metrics data
	bibtex.WriteString(fmt.Sprintf("  sjr = {%f},\n", metrics.SJR))
	bibtex.WriteString(fmt.Sprintf("  avg_citations = {%f},\n", metrics.AvgCitations))
	bibtex.WriteString(fmt.Sprintf("  h_index = {%d},\n", metrics.HIndex))

	// Finalize the BibTeX entry
	output := bibtex.String()
	output = strings.TrimSuffix(output, ",\n") + "\n}\n"

	return output
}

// sortPapersByCitations sorts a list of publications by average citations, using metrics data for sorting criteria.
func sortPapersByCitations(papers []Publication, metrics MetricsDatabase) []Publication {
	// Associate publications with their corresponding metrics if available
	var papersWithMetrics []struct {
		pub     Publication
		metrics JournalMetrics
	}
	for _, paper := range papers {
		metrics, ok := metrics.LookupISSN(paper.ISSN)
		if !ok {
			metrics = JournalMetrics{} // Use a default empty metrics if not found
		}
		papersWithMetrics = append(papersWithMetrics, struct {
			pub     Publication
			metrics JournalMetrics
		}{pub: paper, metrics: metrics})
	}

	// Sort based on average citations
	sort.Slice(papersWithMetrics, func(i, j int) bool {
		return papersWithMetrics[i].metrics.AvgCitations > papersWithMetrics[j].metrics.AvgCitations
	})

	// Extract the sorted publications
	var sortedPapers []Publication
	for _, paper := range papersWithMetrics {
		sortedPapers = append(sortedPapers, paper.pub)
	}

	return sortedPapers
}

func main() {
	// Ensure the program is run with the correct arguments
	if len(os.Args) != 3 {
		log.Printf("Usage: %s <paper xml filename> <impact factor csv>", os.Args[0])
		os.Exit(1)
	}
	xmlFilename := os.Args[1]
	csvFilename := os.Args[2]

	// Read the XML file containing paper information
	xmlData, err := os.ReadFile(xmlFilename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Load journal metrics from the CSV file
	journalDB, err := ReadMetricsCSV(csvFilename)
	if err != nil {
		log.Fatalln(err)
	}

	// Parse the XML data into the OAIPMH structure
	var oaiData OAIPMH
	err = xml.Unmarshal(xmlData, &oaiData)
	if err != nil {
		fmt.Printf("Error parsing XML: %v\n", err)
		return
	}

	// Extract publications from the parsed XML records
	pubs := make([]Publication, 0, len(oaiData.ListRecords.Records))
	for _, record := range oaiData.ListRecords.Records {
		pubs = append(pubs, record.Metadata.Publication)
	}

	// Sort publications by average citations
	pubs = sortPapersByCitations(pubs, journalDB)

	// Print out each publication in BibTeX format
	for _, pub := range pubs {
		issn := pub.ISSN
		metrics, _ := journalDB.LookupISSN(issn)
		fmt.Println(toBibTeX(pub, metrics))
	}
}
