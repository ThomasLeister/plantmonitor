package giphy

/*
 * Giphy API keys needs to be set:
 * export GIPHY_API_KEY="hRqkUNLwdvypZFWfw9IbqWHvDRq5AJrO"
 */

import (
	"fmt"
	"github.com/peterhellberg/giphy"
	"math/rand"
)

type Giphy struct {
	Apiclient *giphy.Client
} 

func (g *Giphy) Init() {
	g.Apiclient = giphy.NewClient()
}

func (g *Giphy) GetGifURL(keywords []string) (string, error) {
	// Load GIF
	res, err := g.Apiclient.Search(keywords)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	random := rand.Intn(len(res.Data))
	url := res.Data[random].Images.Original.URL

	return url, nil
}