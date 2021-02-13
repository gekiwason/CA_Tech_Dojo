package main
 
import (
    "fmt"
    // "log"
    "net/http"
    "crypto/md5"
    "io"
    "math/rand"
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
    ID       uint   `gorm:"primaryKey"`
    Name  string `json:name gorm:"unique"`
    Token  string `json:token`
}

type Character struct {
    ID       uint       `gorm:"primaryKey"`
    Name  string
    Rarity  uint8
}

type GachaDrawRequest struct {
    Times   int
}

func main() {
    r := gin.Default()
    r.POST("/user/create", create)
    r.OPTIONS("/user/create", Options)
    r.GET("/user/get", get)
    r.PUT("/user/update", put)
    r.POST("/gacha/draw", gacha)
    r.Run(":8000")
}

func gacha(c *gin.Context) {
    db := gormConnect()
    db.LogMode(true)
    db.Set("gorm:table_options", "ENGINE=InnoDB")
    db.AutoMigrate(&Character{})

    raritys := map[int]int{
        3: 300,
        2: 1200,
        1: 8500,
    }

    req := GachaDrawRequest{}
    c.BindJSON(&req)
    times := req.Times

    result := []Character{}
    characters := []Character{}
    for i := 1; i <= times; i++ {
        rand.Seed(time.Now().UnixNano())
        random := (rand.Intn(10000))
        probability := 0
        for rarity, rarity_probability := range raritys {
            probability += rarity_probability
            if random <= probability { // 排出レアリティ確定
                db.Where("rarity = ?", rarity).Find(&characters) // 排出レアリティ内からランダムに1枚取得
                num := rand.Intn(len(characters))
                result = append(result, characters[num])
                break
            }
        }
        time.Sleep(1 * time.Second) // seed値が近すぎるとガチャ排出
    }
    c.JSON(http.StatusOK, result)
}

func Options(c *gin.Context) {
    if c.Request.Method != "OPTIONS" {
        c.Next()
    } else {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
        c.Header("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
        c.Header("Content-Type", "application/json")
        c.AbortWithStatus(http.StatusOK)
    }
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
}
