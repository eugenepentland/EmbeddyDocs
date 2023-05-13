package functions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/dslipak/pdf"
	"github.com/k3a/html2text"
	"github.com/nlpodyssey/cybertron/pkg/tasks/textencoding"
	"github.com/tiktoken-go/tokenizer"
)

func Cosine(a []float64, b []float64) (cosine float64, err error) {
	count := 0
	length_a := len(a)
	length_b := len(b)
	if length_a > length_b {
		count = length_a
	} else {
		count = length_b
	}
	sumA := 0.0
	s1 := 0.0
	s2 := 0.0
	for k := 0; k < count; k++ {
		if k >= length_a {
			s2 += math.Pow(float64(b[k]), 2)
			continue
		}
		if k >= length_b {
			s1 += math.Pow(float64(a[k]), 2)
			continue
		}
		sumA += float64(a[k] * b[k])
		s1 += math.Pow(float64(a[k]), 2)
		s2 += math.Pow(float64(b[k]), 2)
	}
	if s1 == 0 || s2 == 0 {
		return 0.0, errors.New("s1 or s2 is zero")
	}
	return sumA / (math.Sqrt(s1) * math.Sqrt(s2)), nil
}

type ReaderAtFromReader struct {
	reader io.Reader
}

func (r *ReaderAtFromReader) ReadAt(p []byte, off int64) (n int, err error) {
	// Check if offset is valid
	if off < 0 {
		return 0, errors.New("negative offset")
	}

	// Create a buffer with offset bytes
	_, err = io.CopyN(ioutil.Discard, r.reader, off)
	if err != nil {
		return 0, err
	}

	// Read len(p) bytes into p
	n, err = r.reader.Read(p)
	if err != nil {
		return n, err
	}

	return n, nil
}

func ConvertToOSFile(rsc io.ReadSeekCloser) (*os.File, error) {
	// Create a temporary file
	f, err := ioutil.TempFile("", "tempfile")
	if err != nil {
		return nil, err
	}

	// Write the contents of rsc to the temporary file
	_, err = io.Copy(f, rsc)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Seek to the beginning of the temporary file
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		f.Close()
		return nil, err
	}

	return f, nil
}

func GetPdfContents(file io.ReadSeekCloser, size int64, enc tokenizer.Codec) (FileContents, error) {
	osFile, err := ConvertToOSFile(file)
	if err != nil {
		return FileContents{}, err
	}
	file.Close()
	defer osFile.Close()
	fi, err := osFile.Stat()
	if err != nil {
		return FileContents{}, err
	}
	fmt.Println(fi)

	r, err := pdf.NewReaderEncrypted(osFile, fi.Size(), nil)
	if err != nil {
		fmt.Println(err)
	}

	var wg sync.WaitGroup

	//Creates a max of 5 workers to read the PDF pages
	pageCount := r.NumPage()
	maxWorkers := 5
	workerChan := make(chan struct{}, maxWorkers)
	resultChan := make(chan Embedding, pageCount)

	for i := 1; i <= pageCount; i++ {
		wg.Add(1)
		workerChan <- struct{}{}
		go func(i int) {
			defer func() {
				<-workerChan
				wg.Done()
			}()
			//Reads the contents of the page
			text, err := r.Page(i).GetPlainText(nil)
			if err != nil {
				fmt.Println(err)
				return
			}
			//Gets the token count
			ids, _, err := enc.Encode(text)
			if err != nil {
				fmt.Println(err)
				return
			}
			text = strings.ReplaceAll(text, "\n", " ")
			//Outputs the data to the result channel
			resultChan <- Embedding{
				Text:       text,
				TokenCount: len(ids),
				PageNumber: i,
			}
		}(i)
	}
	wg.Wait()
	close(resultChan)

	var FileContents FileContents
	for res := range resultChan {
		FileContents.Embeddings = append(FileContents.Embeddings, res)
	}

	return FileContents, nil
}

func GetUrlContents(url string, enc tokenizer.Codec) FileContents {
	//Getes the html in plain text
	res, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	plain := html2text.HTML2Text(string(content))

	//Gets the tokens
	id, _, err := enc.Encode(plain)
	if err != nil {
		fmt.Println(err)
	}
	tokenCount := len(id)

	//Formats the output data
	var FileContents FileContents
	FileContents.Embeddings = append(FileContents.Embeddings, Embedding{Text: plain, PageNumber: 1, TokenCount: tokenCount})
	return FileContents
}

func (Payload *EmbeddingPayload) GetEmbedding(ctx context.Context, m textencoding.Interface) error {
	//poolingStrat := int(bert.MeanPooling)
	fmt.Println("encoding page", Payload.PageContent)
	vec, err := m.Encode(ctx, Payload.PageContent, 1)
	if err != nil {
		fmt.Print("error while embedding page", err)
	}
	vecResp := vec.Vector.Data().F64()
	Payload.Embedding = vecResp
	return nil
}

type Embedding struct {
	Text       string
	TokenCount int
	Embedding  []float32
	Model      string
	PageNumber int
	PageIndex  int
}

type FileContents struct {
	Embeddings []Embedding `json:"embeddings"`
}
