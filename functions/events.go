package functions

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/labstack/echo/v5"
	"github.com/nlpodyssey/cybertron/pkg/models/bert"
	"github.com/nlpodyssey/cybertron/pkg/tasks/textencoding"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/tiktoken-go/tokenizer"
)

func InitializeEmbeddingDB(e *core.ServeEvent) error {
	UpsertFileTable(e.App)
	UpsertEmbeddingTable(e.App)
	return nil
}

type Metadata struct {
	Loc Loc `json:"loc"`
}

type Loc struct {
	PageNumber int `json:"pageNumber"`
	PageIndex  int `json:"pageIndex"`
}

type EmbeddingPayload struct {
	PageContent string    `json:"pageContent"`
	Tokens      int       `json:"tokens"`
	Embedding   []float64 `json:"embedding"`
	Metadata    Metadata  `json:"metadata"`
}

func (e *EmbeddingPayload) SetTokenCount(enc tokenizer.Codec) error {
	//Gets the token count
	ids, _, err := enc.Encode(e.PageContent)
	if err != nil {
		return err
	}
	e.Tokens = len(ids)
	return nil
}

func EmbeddingEndpoint(e *core.ServeEvent, m textencoding.Interface, ctx context.Context) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/:fileId/embedding",
		Handler: func(c echo.Context) error {
			//Gets the payload and checks to make sure its valid
			var payload []EmbeddingPayload
			err := c.Bind(&payload)
			if err != nil {
				fmt.Println("binding error", err)
				return c.String(http.StatusBadRequest, err.Error())
			}

			//Loads the tokenizer
			enc, err := tokenizer.Get(tokenizer.Cl100kBase)
			if err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
			//Breaks the payload into chunks of 150 tokens max & gets the embeddings
			maxTokenCount := 128
			payloadLength := len(payload)
			fmt.Println("Payload Page Length", payloadLength)

			//Loops through all of the payloads
			for i := 0; i < payloadLength; i++ {
				fmt.Println("Page", i+1, "of", payloadLength, "pages")
				payload[i].SetTokenCount(enc)
				sectionCount := (payload[i].Tokens / maxTokenCount) + 1
				//Doesn't need to be split
				if sectionCount == 1 {
					payload[i].Metadata.Loc.PageIndex = 1
					continue
				}
				//When the payload needs to be split
				overlapPercent := 10
				splitTextLength := len(payload[i].PageContent) / sectionCount
				splitTokenCount := payload[i].Tokens / sectionCount

				splitTokenCount += int(splitTokenCount * overlapPercent / 100)
				//Loops through the sections
				for j := 0; j < sectionCount; j++ {
					startPos := j * splitTextLength
					endPos := (j + 1) * splitTextLength
					isFirstSection := j == 0
					isLastSection := j == sectionCount-1
					isMiddleSection := !isFirstSection && !isLastSection

					//Adds 20 percent to the end if its the first item
					if isFirstSection {
						endPos += int(endPos * overlapPercent / 100)
						//Adds 20 percent to the beginning and end if its not the first or last item
					} else if isMiddleSection {
						startPos -= int(startPos * overlapPercent / 100)
						endPos += int(endPos * overlapPercent / 100)
						splitTokenCount += int(splitTokenCount * overlapPercent / 100)
						//Adds 20 percent to the beginning if its the last item
					} else if isLastSection {
						startPos -= int(startPos * overlapPercent / 100)
					}

					subText := payload[i].PageContent[startPos:endPos]
					if j != sectionCount-1 {
						embedding := EmbeddingPayload{
							PageContent: subText,
							Tokens:      splitTokenCount,
							Metadata: Metadata{
								Loc: Loc{
									PageNumber: payload[i].Metadata.Loc.PageNumber,
									PageIndex:  j + 1,
								},
							},
						}
						payload = append(payload, embedding)
					} else {
						payload[i].PageContent = subText
						payload[i].Metadata.Loc.PageIndex = j + 1
						payload[i].Tokens = splitTokenCount
					}

				}

			}
			for i := range payload {
				err := payload[i].GetEmbedding(ctx, m)
				if err != nil {
					return c.String(http.StatusBadRequest, err.Error())
				}
			}
			//Loads the embedding collection
			collection, err := e.App.Dao().FindCollectionByNameOrId("embeddings")
			if err != nil {
				return err
			}
			e.App.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				for i := range payload {
					record := models.NewRecord(collection)
					record.Set("token_count", payload[i].Tokens)
					record.Set("file", c.PathParam("fileId"))
					record.Set("page_number", payload[i].Metadata.Loc.PageNumber)
					record.Set("similarity_score", GetSimilarityScore(payload[i], payload))
					record.Set("text", payload[i].PageContent)
					record.Set("embedding", payload[i].Embedding)
					record.Set("page_index", payload[i].Metadata.Loc.PageIndex)
					if err := txDao.SaveRecord(record); err != nil {
						return err
					}
				}
				return nil
			})

			return nil
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(e.App),
			//apis.RequireRecordAuth("embeddings"),
		},
	})
	return nil
}

func GetSimilarityScore(Embedding EmbeddingPayload, EmbeddingList []EmbeddingPayload) float64 {
	var score float64 = 0
	for i := 0; i < len(EmbeddingList); i++ {
		similarity, err := Cosine(Embedding.Embedding, EmbeddingList[i].Embedding)
		if err != nil {
			fmt.Println(err)
		}
		score += similarity
	}
	return score / float64(len(EmbeddingList))
}

func VectorSearch(e *core.RecordsListEvent, m textencoding.Interface, ctx context.Context) error {
	searchQuery := e.HttpContext.QueryParam("search")
	if searchQuery == "" {
		return nil
	}
	poolingStrat := int(bert.MeanPooling)
	result, err := m.Encode(ctx, searchQuery, poolingStrat)
	if err != nil {
		panic(err)
	}
	queryVector := result.Vector.Data().F64()

	for _, v := range e.Records {
		var recordVector []float64
		err := v.UnmarshalJSONField("embedding", &recordVector)
		if err != nil {
			fmt.Println(err)
		}
		similarity, err := Cosine(queryVector, recordVector)
		if err != nil {
			fmt.Println(err)
		}
		v.Set("similarity", similarity)
	}
	//Sorts by similarity
	sort.Slice(e.Records, func(i, j int) bool {
		return e.Records[i].GetFloat("similarity") > e.Records[j].GetFloat("similarity")
	})

	//Gets as many documents as possible without going over the token limit
	maxDocumentCount := 10
	documentCount := 0
	//pageDocs := map[int]string{}
	e.Records = e.Records[:maxDocumentCount]
	for i, record := range e.Records {
		if documentCount <= maxDocumentCount {
			documentCount += 1
			fmt.Println(documentCount, maxDocumentCount)
			text := record.GetString("text")
			similarity := record.GetFloat("similarity")

			bestSimilarity, bestText, loopCount := SimilarityShrink(m, queryVector, similarity, text, 55, 0)
			fmt.Println("Best Similarity:", bestSimilarity, "Starting Simiarity:",similarity,"Page Number:", record.GetInt("page_number"),"Loop Count", loopCount,"\n", "Best Text:", bestText)
			fmt.Println("")
			e.Records[i].Set("text", bestText)
			e.Records[i].Set("similarity", bestSimilarity)
			//Does a sliding window through the document to see if it can find better context

			//pageDocs[record.GetInt("page_number")] += record.GetString("text")
		}
	}
	e.Result.Items = e.Records[:maxDocumentCount]
	//Sorts by similarity
	sort.Slice(e.Records, func(i, j int) bool {
		return e.Records[i].GetFloat("similarity") > e.Records[j].GetFloat("similarity")
	})
	e.Result.Items = e.Records[:5]
	fmt.Println("Vector Search: Relevant Documents:", "Total Tokens:")

	return nil

}
func SimilarityShrink(m textencoding.Interface, queryVector []float64, baseSimilarity float64, text string, overlapPercentage int, loopCount int) (bestSimilarity float64, bestText string, finishLoopCount int) {
	//Creates two subtexts
	return baseSimilarity, text, loopCount
	//The left side is the first 75% of the text and the right side is the last 75% of the text
	textLength := len(text)
	subTextLeftEnd := textLength * overlapPercentage / 100
	subTextRightStart := textLength * (100 - overlapPercentage) / 100
	subTextLeft := text[:subTextLeftEnd]
	subTextRight := text[subTextRightStart:]
	//Gets the embeddings for both subtexts
	vecLeft, err := m.Encode(context.Background(), subTextLeft, 1)
	if err != nil {
		fmt.Print("error while embedding page", err)
	}
	vecLeftResp := vecLeft.Vector.Data().F64()
	vecRight, err := m.Encode(context.Background(), subTextRight, 1)
	if err != nil {
		fmt.Print("error while embedding page", err)
	}
	vecRightResp := vecRight.Vector.Data().F64()
	//Calculates the cosine similarity for both subtexts
	similarityLeft, err := Cosine(vecLeftResp, queryVector)
	if err != nil {
		fmt.Print("error while calculating cosine similarity", err)
	}
	similarityRight, err := Cosine(vecRightResp, queryVector)
	if err != nil {
		fmt.Print("error while calculating cosine similarity", err)
	}
	//Gets the higher of two cosine similarities
	if similarityLeft > similarityRight {
		bestSimilarity = similarityLeft
		bestText = subTextLeft
	} else {
		bestSimilarity = similarityRight
		bestText = subTextRight
	}

	if bestSimilarity > baseSimilarity {
		return SimilarityShrink(m, queryVector, bestSimilarity, bestText, overlapPercentage, loopCount + 1)
	} else {
		return baseSimilarity, text, loopCount
	}
	//return baseSimilarity - (baseSimilarity * shrinkFactor / 100)
}
