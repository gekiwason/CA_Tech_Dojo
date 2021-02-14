package main

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `json:name gorm:"unique"`
	Token string `json:token`
	Coin  uint   `json:coin`
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

type UserCreateRequest struct {
	Name string `json:"name"`
}

type UserCreateResponse struct {
	Token string `json:"token"`
}

type UserGetResponse struct {
	Name string `json:"name"`
}

type UserUpdateRequest struct {
	Name string `json:"name"`
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

type CharacterSellRequest struct {
	UserCharacterID string `json:"userCharacterID"`
}

type CharacterSellResponse struct {
	UserCoin string `json:"userCoin"`
}

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
	return db
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	db := gormConnect()
	db.Set("gorm:table_options", "ENGINE=InnoDB")
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Character{})
	db.AutoMigrate(&UserCharacter{})

	r := gin.Default()

	// CORS 対応
	r.Use(CORSMiddleware())

	r.POST("/user/create", create)
	r.GET("/user/get", get)
	r.PUT("/user/update", put)
	r.POST("/gacha/draw", gacha)
	r.GET("/character/list", list)
	r.DELETE("/character/sell", sell)
	r.Run(":8000")
}

func create(c *gin.Context) {
	db := gormConnect()

	req := UserCreateRequest{}
	c.BindJSON(&req)

	user := User{}

	salt := time.Now().Unix()
	m5 := md5.New()
	m5.Write([]byte(user.Name))
	m5.Write([]byte(fmt.Sprint(salt)))
	token := fmt.Sprintf("%x", m5.Sum(nil))

	user.Name = req.Name
	user.Token = token

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res := UserCreateResponse{}
	res.Token = user.Token

	c.JSON(http.StatusOK, gin.H{
		"token": res.Token,
	})
}

func get(c *gin.Context) {
	db := gormConnect()

	user := User{}
	token := c.Request.Header.Get("x-token")

	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res := UserGetResponse{}
	res.Name = user.Name
	c.JSON(http.StatusOK, gin.H{
		"name": res.Name,
	})
}

func put(c *gin.Context) {
	db := gormConnect()

	user := User{}
	req := UserUpdateRequest{}
	token := c.Request.Header.Get("x-token")

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user.Name = req.Name
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.String(http.StatusOK, "successed!")
}

func gacha(c *gin.Context) {
	db := gormConnect()

	user := User{}
	token := c.Request.Header.Get("x-token")
	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	raritys := map[int]int{
		3: 300,  //SSR
		2: 1200, //SR
		1: 8500, //R
	}

	req := GachaDrawRequest{}
	c.BindJSON(&req)
	times := req.Times

	result := Character{}
	characters := []Character{}
	gachaResults := []GachaResult{}

	for i := 1; i <= times; i++ {

		userCharacter := UserCharacter{}
		userCharacter.UserID = user.ID

		rand.Seed(time.Now().UnixNano())
		random := (rand.Intn(10000))
		probability := 0

		gachaResult := GachaResult{}
		for rarity, rarityProbability := range raritys {
			probability += rarityProbability
			if random <= probability { // 排出レアリティ確定
				db.Where("rarity = ?", rarity).Find(&characters) // 排出レアリティ内のキャラ一覧を取得
				num := rand.Intn(len(characters))
				result = characters[num]

				userCharacter.CharacterID = result.ID
				db.Create(&userCharacter)

				gachaResult.CharacterID = fmt.Sprint(result.ID)
				gachaResult.Name = result.Name
				gachaResults = append(gachaResults, gachaResult)

				break
			}
		}

		time.Sleep(1 * time.Second) // seed値をずらしガチャ排出を均一に
	}

	c.JSON(http.StatusOK, gin.H{
		"results": gachaResults,
	})
}

func list(c *gin.Context) {
	db := gormConnect()
	user := User{}
	userCharacters := []UserCharacter{}

	token := c.Request.Header.Get("x-token")
	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := db.Where("user_id = ?", user.ID).Find(&userCharacters).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res := []CharacterListResponse{}
	for _, v := range userCharacters {
		character := Character{}
		characterForRes := CharacterListResponse{}
		if err := db.First(&character, "id = ?", v.CharacterID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
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

func sell(c *gin.Context) {
	db := gormConnect()

	user := User{}
	token := c.Request.Header.Get("x-token")
	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	req := CharacterSellRequest{}
	c.BindJSON(&req)
	userCharacter := UserCharacter{}
	if err := db.Where("id = ? AND user_id = ?", req.UserCharacterID, user.ID).First(&userCharacter).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	character := Character{}
	if err := db.First(&character, "id = ?", userCharacter.CharacterID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	switch character.Rarity {
	case 3:
		db.Delete(&userCharacter)
		user.Coin += 10000
	case 2:
		db.Delete(&userCharacter)
		user.Coin += 3000
	case 1:
		db.Delete(&userCharacter)
		user.Coin += 500
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unknown rarity",
		})
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res := CharacterSellResponse{}
	res.UserCoin = fmt.Sprint(user.Coin)

	c.JSON(http.StatusOK, gin.H{
		"userCoin": res.UserCoin,
	})
}
