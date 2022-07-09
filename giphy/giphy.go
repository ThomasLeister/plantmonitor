package giphy

import (
	"fmt"
	"github.com/sanzaru/go-giphy"
)

type Giphy struct {
	Apiclient *libgiphy.Giphy
} 

func (g *Giphy) Init(api_key string) {
	g.Apiclient = libgiphy.NewGiphy(api_key)
}

func (g *Giphy) GetGifURL(keywords string) (string, error) {
    dataRandom, err := g.Apiclient.GetRandom(keywords)
    if err != nil {
        fmt.Println("error:", err)
		return "", err
    }
    
	gifUrl := dataRandom.Data.Images.Original.Mp4
    fmt.Printf("GIF URL: %+v\n", gifUrl)

	return gifUrl, nil
}