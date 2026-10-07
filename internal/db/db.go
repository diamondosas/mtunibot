package db

import (
	"log"
	"os"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var usersDB *gorm.DB
var notesDB *gorm.DB

type User struct {
	ID         uint   `gorm:"primaryKey"`
	TelegramID int64  `gorm:"uniqueIndex;not null"`
	Username   string `gorm:"size:255"`
	Level      int    `gorm:"column:level"`
	College    string `gorm:"size:255"`
	Department string `gorm:"size:255"`
}

type Note struct {
	ID         uint   `gorm:"primaryKey"`
	CourseCode string `gorm:"index;size:20"`
	Title      string `gorm:"size:255"`
	College    string `gorm:"size:50"`
	Department string `gorm:"size:100"`
	Level      string `gorm:"size:20"`
	FileID     string `gorm:"size:255"`
	FileName   string `gorm:"size:255"`
	FileHash   string `gorm:"index;size:32"` // MD5 content hash
	Tags       string `gorm:"type:text"` // Comma-separated search tags
}

func InitDB() error {
	var err error
	_, err = os.Stat("db/")
	if os.IsNotExist(err) {
		err = os.Mkdir("db", 0777)
		if err != nil {
			log.Println("Could not create db dir:", err)
		}
	}

	usersDB, err = gorm.Open(sqlite.Open("db/users.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect User to database:", err)
	}

	notesDB, err = gorm.Open(sqlite.Open("db/notes.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect Notes to database:", err)
	}

	err = usersDB.AutoMigrate(&User{})
	if err != nil {
		log.Println("Could not initialise User table:", err)
	}

	err = notesDB.AutoMigrate(&Note{})
	if err != nil {
		log.Println("Could not initialise Note table:", err)
	}

	log.Println("Database initialized successfully!")
	return nil
}

func DoesUserExist(telegramID int64, username string) bool {
	var user User
	result := usersDB.Where("telegram_id = ?", telegramID).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		err := SaveUser(telegramID, username)
		if err != nil {
			log.Println("Could not save user", err)
		}
		return false
	}
	return true
}

func SaveUser(telegramID int64, username string) error {
	newUser := User{
		TelegramID: telegramID,
		Username:   username,
	}
	err := usersDB.Create(&newUser).Error
	if err != nil {
		log.Println("Could not create new user:", err)
		return err
	}
	log.Println(username, "added to usersDB")
	return nil
}

func GetUser(telegramID int64) (*User, error) {
	var user User
	err := usersDB.Where("telegram_id = ?", telegramID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UpdateUserProfile(telegramID int64, college, department string, level int) error {
	var user User
	res := usersDB.Where("telegram_id = ?", telegramID).First(&user)
	if res.Error == gorm.ErrRecordNotFound {
		user = User{
			TelegramID: telegramID,
			College:    college,
			Department: department,
			Level:      level,
		}
		return usersDB.Create(&user).Error
	}
	return usersDB.Model(&user).Updates(map[string]interface{}{
		"college":    college,
		"department": department,
		"level":      level,
	}).Error
}

func SaveNote(note *Note) error {
	return notesDB.Create(note).Error
}

func GetNoteByID(id uint) (*Note, error) {
	var note Note
	err := notesDB.First(&note, id).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func SearchNotes(query string) ([]Note, error) {
	var notes []Note
	q := "%" + strings.TrimSpace(query) + "%"
	err := notesDB.Where("course_code LIKE ? OR title LIKE ? OR tags LIKE ? OR department LIKE ?", q, q, q, q).Find(&notes).Error
	return notes, err
}

func GetNotesByDeptAndLevel(dept string, level string) ([]Note, error) {
	var notes []Note
	lvlLike := "%" + strings.TrimSpace(level) + "%"
	err := notesDB.Where("department = ? AND (level = ? OR level LIKE ?)", dept, level, lvlLike).Find(&notes).Error
	return notes, err
}

func GetNoteByHash(hash string) (*Note, error) {
	if hash == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var note Note
	err := notesDB.Where("file_hash = ?", hash).First(&note).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func GetCoursesByDeptAndLevel(dept string, level string) ([]string, error) {
	var courses []string
	lvlLike := "%" + strings.TrimSpace(level) + "%"
	err := notesDB.Model(&Note{}).
		Where("department = ? AND (level = ? OR level LIKE ?)", dept, level, lvlLike).
		Distinct("course_code").
		Pluck("course_code", &courses).Error
	return courses, err
}

func GetNotesByCourse(dept string, level string, courseCode string) ([]Note, error) {
	var notes []Note
	lvlLike := "%" + strings.TrimSpace(level) + "%"
	err := notesDB.Where("department = ? AND (level = ? OR level LIKE ?) AND course_code = ?", dept, level, lvlLike, courseCode).Find(&notes).Error
	return notes, err
}
