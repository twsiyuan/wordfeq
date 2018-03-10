# wordfreq

[Text corpus](https://en.wikipedia.org/wiki/Text_corpus) calculation in Golang. 
Supports Chinese, English.

This work is a derivative of [wordfreq](https://github.com/timdream/wordfreq/) by [Timothy Guan-tin Chien](http://timc.idv.tw/).

## Install

With a [correctly configured](https://golang.org/doc/install#testing) Go toolchain:

```sh
go get -u github.com/twsiyuan/wordfreq
```

## Simple Example

```go
import(
   "github.com/twsiyuan/wordfreq"
)

func main(){
   wfreq, _ := wordfreq.New(wordfreq.Options{})
   tlist := wfreq.Process("text")  // Term list
}
```

Available options in ```wordfreq.Options```:

- ```Languages```: Array of keywords to specify languages to process. Available keywords are ```chinese```, ```english```. Default to both.
- ```StopWordSets```: Array of keywords to specify the built-in set of stop words to exclude in the count. Available: ```cjk```, ```english1```, and ```english2```. Default to all.
- ```StopWords```: Array of words/phrases to exclude in the count. Case insensitive. Default to empty.
- ```MinimumCount```: Minimal count required to be included in the returned list. Default to ```2```.
- ```NoFilterSubstring```: (Chinese language only) No filter out the recounted substring. Default to ```false```.
- ```MaxiumPhraseLength```: (Chinese language only) Maxium length to consider a phrase. Default to ```8```.