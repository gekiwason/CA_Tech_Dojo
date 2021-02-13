package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func gormConnect() *gorm.DB {
	DBMS := "mysql"
	USER := "root"
	PASS := "root"
	PROTOCOL := "tcp(localhost:3306)"
	DBNAME := "CA_Tech_Dojo"
	CONNECT := USER + ":" + PASS + "@" + PROTOCOL + "/" + DBNAME
	db, err := gorm.Open(DBMS, CONNECT)

	if err != nil {
		panic(err.Error())
	}
	fmt.Println("db connected: ", &db)
	return db
}

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `json:name gorm:"unique"`
	Token string `json:token`
}

type Character struct {
	ID     uint `gorm:"primaryKey"`
	Name   string
	Rarity uint8
}

type UserCharacter struct {
	ID          uint `gorm:"primaryKey"`
	UserID      uint
	CharacterID uint
}

type GachaDrawRequest struct {
	Times int
}

type GachaResult struct {
	CharacterID string `json:"characterID"`
	Name        string `json:"name"`
}

type CharacterListResponse struct {
	UserCharacterID string `json:"userCharacterID"`
	CharacterID     string `json:"characterID"`
	Name            string `json:"name"`
}

func main() {
	r := gin.Default()
	r.POST("/user/create", create)
	r.GET("/user/get", get)
	r.PUT("/user/update", put)
	r.POST("/gacha/draw", gacha)
	r.GET("/character/list", list)
	r.Run(":8000")
}

func list(c *gin.Context) {
	db := gormConnect()
	db.LogMode(true)
	user := User{}
	userCharacters := []UserCharacter{}

	token := c.Request.Header.Get("x-token")
	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		c.String(http.StatusBadRequest, "Request is failed: "+err.Error())
		return
	}

	if err := db.Where("user_id = ?", user.ID).Find(&userCharacters).Error; err != nil {
		c.String(http.StatusBadRequest, "Request is failed: "+err.Error())
		return
	}

	res := []CharacterListResponse{}
	for _, v := range userCharacters {
		character := Character{}
		characterForRes := CharacterListResponse{}
		log.Println(v.CharacterID)
		if err := db.First(&character, "id = ?", v.CharacterID).Error; err != nil {
			c.String(http.StatusBadRequest, "Request is failed: "+err.Error())
			return
		}

		characterForRes.UserCharacterID = fmt.Sprint(v.ID)
		characterForRes.CharacterID = fmt.Sprint(character.ID)
		characterForRes.Name = character.Name

		res = append(res, characterForRes)
	}

	c.JSON(http.StatusOK, gin.H{
		"characters": res,
	})
}

func gacha(c *gin.Context) {
	db := gormConnect()
	db.LogMode(true)
	db.Set("gorm:table_options", "ENGINE=InnoDB")
	db.AutoMigrate(&Character{})
	db.AutoMigrate(&UserCharacter{})

	user := User{}
	token := c.Request.Header.Get("x-token")
	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		c.String(http.StatusBadRequest, "Request is failed: "+err.Error())
		return
	}

	raritys := map[int]int{
		3: 300,
		2: 1200,
		1: 8500,
	}

	req := GachaDrawRequest{}
	c.BindJSON(&req)
	times := req.Times

	result := Character{}
	characters := []Character{}
	results := []GachaResult{}

	for i := 1; i <= times; i++ {

		userCharacter := UserCharacter{}
		userCharacter.UserID = user.ID

		rand.Seed(time.Now().UnixNano())
		random := (rand.Intn(10000))
		probability := 0

		gachaResult := GachaResult{}
		for rarity, rarity_probability := range raritys {
			probability += rarity_probability
			if random <= probability { // 排出レアリティ確定
				db.Where("rarity = ?", rarity).Find(&characters) // 排出レアリティ内のキャラ一覧を取得
				num := rand.Intn(len(characters))
				result = characters[num]

				userCharacter.CharacterID = result.ID
				db.Create(&userCharacter)

				gachaResult.CharacterID = fmt.Sprint(result.ID)
				gachaResult.Name = result.Name
				results = append(results, gachaResult)

				log.Printf("%v", results)
				break
			}
		}

		time.Sleep(1 * time.Second) // seed値をずらしガチャ排出を均一に
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}

func create(c *gin.Context) {
	db := gormConnect()
	db.LogMode(true)
	db.Set("gorm:table_options", "ENGINE=InnoDB")
	db.AutoMigrate(&User{})

	user := User{}
	c.BindJSON(&user)

	token := "q3489fj9q0348nfq034mf"
	token = user.Name + token
	h := md5.New()
	io.WriteString(h, token)
	s := fmt.Sprintf("%x", h.Sum(nil))

	user.Token = s

	db.NewRecord(user)
	db.Create(&user)
	if db.NewRecord(user) == false {
		c.JSON(http.StatusOK, gin.H{
			"token": user.Token,
		})
	}
}

func get(c *gin.Context) {
	db := gormConnect()
	db.LogMode(true)

	user := User{}
	token := c.Request.Header.Get("x-token")

	db.Where("token = ?", token).First(&user)
	c.JSON(http.StatusOK, gin.H{
		"name": user.Name,
	})
}

func put(c *gin.Context) {
	db := gormConnect()
	db.LogMode(true)

	user := User{}
	token := c.Request.Header.Get("x-token")

	data := User{}
	if err := c.BindJSON(&data); err != nil {
		c.String(http.StatusBadRequest, "Request is failed: "+err.Error())
	}

	db.Where("token = ?", token).First(&user).Updates(&data)
	c.Status(http.StatusOK)
}
