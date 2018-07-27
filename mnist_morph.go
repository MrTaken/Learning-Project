package main

import (
	"fmt"
	"flag"
	"net/http"
	"log"
	"github.com/gorilla/websocket"
	b64 "encoding/base64"
	"time"
	"math/rand"

	"sort"
	"os/exec"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{} // use default options

func random(min, max int) int {
	return rand.Intn(max - min) + min
}

func sendImg(c *websocket.Conn){
	for{
		time.Sleep(time.Microsecond*33)
		encoded:=b64.StdEncoding.EncodeToString(imgs[random(1, 10000)])
		c.WriteMessage(1, []byte(encoded))
	}
}
func sendFrames(c *websocket.Conn,lastid int,usedids []int)(int,[]int){
	gknn,usedids:=generateFrames(lastid,usedids)
	var acc []byte
	for j := 0; j < len(gknn); j++{
		acc = append(acc,imgs[gknn[j]]...)
	}
	encoded:=b64.StdEncoding.EncodeToString(acc)
	c.WriteMessage(1, []byte(encoded))
	return gknn[len(gknn)-1],usedids
}
func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	//go sendImg(c)
	nines:=getIdArray(mainnum)
	var usedids []int
	lastid:=nines[random(0, len(nines)-1)]
	lastid,usedids=sendFrames(c,lastid,usedids)
	for {
		_, _, err := c.ReadMessage()

		if err != nil {
			log.Println("read:", err)
			break
		}
		//log.Printf("recv: %s", message)
		lastid,usedids=sendFrames(c,lastid,usedids)
		//encoded:=b64.StdEncoding.EncodeToString(imgs[5])
		//err = c.WriteMessage(mt, []byte(encoded))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "home.html")
	//homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

var imgs []RawImage
var labels []Label
func getIdArray(num int) []int{
	var result []int
	for i, lnum := range labels {
		if num == int(lnum) {
			result=append(result, i)
		}
	}
	return result
}
func calcScore(img1 RawImage,img2 RawImage) int{
	var result int
	for i,_:=range img1{
		diff:=int(img1[i])-int(img2[i])
		if diff<0{diff=-diff}
		result+=diff
	}
	return result

}
func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
type line struct{
	id int
	score int
}
type byScore []line
func (a byScore) Len() int{ return len(a) }
func (a byScore) Less(i, j int) bool { return a[i].score < a[j].score }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func getKNN(top []topline,k int, id int) []int{
	var result []int

	var topscore []line
	for i,val:=range top{
		if contains(val.comb,id){
			topscore=append(topscore, line{i,val.score})
		}
	}
	sort.Sort(byScore(topscore))
	for _,val:=range topscore[:k]{
		result=append(result,val.id)
	}
	return  result
}
type topline struct{
	comb []int
	score int
}


func line2id(line topline,exclude int)int{
	if line.comb[0]==exclude{
		return line.comb[1]
	}else if line.comb[1]==exclude{
		return line.comb[0]
	}
	panic(fmt.Sprintf("no id in combination"))
	return 0
}
var top []topline
func generateFrames(lastid int,usedids []int)([]int,[]int){

	var knn []int
	frames:=60
	//println(len(usedids))
	knn=append(knn,lastid)
	for i := 0; i < frames-1; i++ {
		ids:=getKNN(top,frames*4+1,knn[len(knn)-1])
		for _, v := range ids {
			id:=line2id(top[v],knn[len(knn)-1])
			if int(labels[id])!=mainnum{
				panic(fmt.Sprintf("not a needed number"))
			}
			if !contains(usedids,id){
				knn=append(knn, id)
				if len(usedids)>=frames*4{usedids=usedids[1:]}
				usedids=append(usedids,id)
				break
			}
		}
	}
	return knn,usedids
}

var mainnum int=9;
func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	rows, cols,local_imgs,err:=ReadImageFile("t10k-images-idx3-ubyte.gz")
	local_labels,err:=ReadLabelFile("t10k-labels-idx1-ubyte.gz")
	fmt.Println(rows, cols,len(local_imgs),len(local_labels),err)
	imgs=local_imgs
	labels=local_labels
	nines:=getIdArray(mainnum)

	for i := 0; i < len(nines); i++ {
		for j := i + 1; j < len(nines); j++ {
			score := calcScore(imgs[nines[i]], imgs[nines[j]])
			comb := []int{nines[i], nines[j]}
			top = append(top, topline{comb, score})
		}
	}

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	c:=exec.Command("cmd", "/C","start http://localhost:8080/")
	if err := c.Run(); err != nil {
		fmt.Println("Error: ", err)
	}
	log.Fatal(http.ListenAndServe(*addr, nil))

}