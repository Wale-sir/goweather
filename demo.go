package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	http.HandleFunc("/weather", weatherHandler)
	fmt.Println("Server running on port 8080")
	http.ListenAndServe("0.0.0.0:8080", nil)
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handlePost(w, r)
	case http.MethodGet:
		handleGet(w, r)
	default:
		http.Error(w, "只允许GET和POST方法", http.StatusMethodNotAllowed)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	// 读取地区名称和代码的映射关系
	areaCodes, err := readAreaCodes("tqdm.txt")
	if err != nil {
		http.Error(w, "读取文件失败", http.StatusInternalServerError)
		return
	}

	// 解析请求体中的JSON数据
	var requestData map[string]string
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求正文失败", http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, "解析JSON数据失败", http.StatusBadRequest)
		return
	}

	// 获取地区名称
	areaName := requestData["area"]
	areaCode, ok := areaCodes[areaName]
	if !ok {
		http.Error(w, "找不到区号", http.StatusNotFound)
		return
	}

	// 获取天气信息
	weatherInfo, err := getWeather(areaName, areaCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取天气信息失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(weatherInfo)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	// 从查询参数中获取地区名称
	areaName := r.URL.Query().Get("area")
	if areaName == "" {
		http.Error(w, "缺少查询参数: area", http.StatusBadRequest)
		return
	}

	// 读取地区名称和代码的映射关系
	areaCodes, err := readAreaCodes("tqdm.txt")
	if err != nil {
		http.Error(w, "读取文件失败", http.StatusInternalServerError)
		return
	}

	areaCode, ok := areaCodes[areaName]
	if !ok {
		http.Error(w, "找不到区号", http.StatusNotFound)
		return
	}

	// 获取天气信息
	weatherInfo, err := getWeather(areaName, areaCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取天气信息失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(weatherInfo)
}

func readAreaCodes(filename string) (map[string]string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	areaCodes := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		// 去除行末的回车符
		line = strings.TrimSuffix(line, "\r")

		parts := strings.Split(line, " = ")
		if len(parts) == 2 {
			areaCodes[parts[0]] = parts[1]
		}
	}

	return areaCodes, nil
}

func getWeather(cityName, dm string) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://www.weather.com.cn/weather1d/%s.shtml#search", dm)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求出错: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应出错: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("解析HTML出错: %v", err)
	}

	weatherInfo := make(map[string]interface{})
	weatherInfo["city"] = cityName

	// 提取<input>标签的值
	inputValue := doc.Find(`#hidden_title`).AttrOr("value", "")

	// 使用正则表达式提取信息
	reDate := regexp.MustCompile(`(\d+月\d+日\d+时)`)
	reWeather := regexp.MustCompile(`(周.)\s*([\p{Han}\w]+转?[\p{Han}\w]*)`)
	reTemperature := regexp.MustCompile(`(\d+/\d+°C)`)

	date := reDate.FindString(inputValue)
	weather := reWeather.FindStringSubmatch(inputValue)
	temperature := reTemperature.FindString(inputValue)

	weatherInfo["date"] = date
	if len(weather) > 2 {
		weatherInfo["weekday"] = weather[1]
		weatherInfo["weather"] = weather[2]
	}
	weatherInfo["temperature"] = temperature

	return weatherInfo, nil
}
