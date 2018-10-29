package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/improbable-eng/thanos/pkg/block"
	"github.com/minio/minio-go"
	"github.com/sbueringer/thanos-inspect/pkg/table"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var (
	configFile   = flag.String("config-file", os.Getenv("HOME")+"/.mc/config.json", "minio-style config file")
	overwriteURL = flag.String("overwrite-url", "localhost:9000", "minio-style config file")
	region       = flag.String("region", "c01", "region to inspect")
	bucket       = flag.String("bucket", "prometheus", "bucket to inspect")

	replica = flag.String("replica", "", "filter by replica")
	sortBy  = flag.String("sort-by", "FROM,UNTIL", "sort by columns")
)

func main() {
	flag.Parse()

	c, err := parseConfig()
	if err != nil {
		panic(err)
	}

	blockMetas, err := downloadMetadata(c)
	if err != nil {
		panic(err)
	}

	printTable(blockMetas)
}

type config struct {
	Version string          `json:"version"`
	Hosts   map[string]host `json:"hosts"`
}

type host struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`
	Lookup    string `json:"lookup"`
}

func parseConfig() (*config, error) {
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	c := &config{}
	err = json.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func downloadMetadata(c *config) ([]*block.Meta, error) {
	host := c.Hosts[*region]

	secure := true
	url := strings.TrimPrefix(host.URL, "https://")
	if *overwriteURL != "" {
		secure = false
		url = *overwriteURL
	}

	client, err := minio.New(url, host.AccessKey, host.SecretKey, secure)
	if err != nil {
		return nil, fmt.Errorf("error creating minio client: %v", err)

	}

	doneCh := make(chan struct{}, 1)
	defer close(doneCh)

	var objInfos []minio.ObjectInfo
	for obj := range client.ListObjects(*bucket, "", true, doneCh) {
		if obj.Err != nil {
			return nil, fmt.Errorf("error reading objects from %s/%s: %v", *bucket, "", obj.Err)
		}
		objInfos = append(objInfos, obj)
	}

	var blockMetas []*block.Meta
	for _, objInfo := range objInfos {
		if strings.Contains(objInfo.Key, "meta.json") {
			obj, err := client.GetObject(*bucket, objInfo.Key, minio.GetObjectOptions{})
			if err != nil {
				fmt.Printf("ERROR: cannot download block: %s\n", objInfo.Key)
			}

			blockMeta := &block.Meta{}
			err = json.NewDecoder(obj).Decode(blockMeta)
			if err != nil {
				fmt.Printf("ERROR: cannot download block: %s\n", objInfo.Key)
			}
			blockMetas = append(blockMetas, blockMeta)
		}
	}
	return blockMetas, nil
}

func printTable(blockMetas []*block.Meta) error {

	header := []string{"ULID", "FROM", "UNTIL", "RANGE", "UNTIL-COMP", "~SIZE", "#SERIES", "#SAMPLES", "#CHUNKS", "COMP-LEVEL", "COMP-FAILED", "REPLICA", "RESOLUTION", "SOURCE"}

	var lines [][]string
	p := message.NewPrinter(language.English)

	for _, blockMeta := range blockMetas {
		if *replica != "" && *replica != blockMeta.Thanos.Labels["replica"] {
			continue
		}

		var line []string
		line = append(line, blockMeta.ULID.String())
		line = append(line, time.Unix(blockMeta.MinTime/1000, 0).Format("02-01-2006 15:04:05"))
		line = append(line, time.Unix(blockMeta.MaxTime/1000, 0).Format("02-01-2006 15:04:05"))
		timeRange := time.Duration((blockMeta.MaxTime - blockMeta.MinTime) * 1000 * 1000)
		line = append(line, timeRange.String())
		untilComp := "-"
		if blockMeta.Thanos.Downsample.Resolution == 0 { // data currently raw, downsample if range >= 40 hours
			untilComp = (time.Duration(40*60*60*1000*time.Millisecond) - timeRange).String()
		}
		if blockMeta.Thanos.Downsample.Resolution == 5*60*1000 { // data currently 5m resolution, downsample if range >= 10 days
			untilComp = (time.Duration(10*24*60*60*1000*time.Millisecond) - timeRange).String()
		}
		line = append(line, untilComp)
		line = append(line, p.Sprintf("%0.2fMiB", (float64(blockMeta.Stats.NumSamples)*1.07)/(1024*1024)))
		line = append(line, p.Sprintf("%d", blockMeta.Stats.NumSeries))
		line = append(line, p.Sprintf("%d", blockMeta.Stats.NumSamples))
		line = append(line, p.Sprintf("%d", blockMeta.Stats.NumChunks))
		line = append(line, p.Sprintf("%d", blockMeta.Compaction.Level))
		line = append(line, p.Sprintf("%t", blockMeta.Compaction.Failed))
		line = append(line, blockMeta.Thanos.Labels["replica"])
		line = append(line, time.Duration(blockMeta.Thanos.Downsample.Resolution * 1000000).String())
		line = append(line, string(blockMeta.Thanos.Source))

		lines = append(lines, line)
	}

	var sortByColNum []int
	for _, col := range strings.Split(*sortBy, ",") {
		index := getIndex(header, col)
		if index == -1 {
			return fmt.Errorf("column %s not found", col)
		}
		sortByColNum = append(sortByColNum, index)
	}

	output, err := table.ConvertToTable(table.Table{Header: header, Lines: lines, SortIndices: sortByColNum, Output: "markdown"})
	if err != nil {
		return err
	}
	fmt.Printf(output)
	return nil
}

func getIndex(cols []string, s string) int {
	for i, col := range cols {
		if col == s {
			return i
		}
	}
	return -1
}
