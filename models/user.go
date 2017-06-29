package models

import (
	"encoding/json"
	"fmt"
	"github.com/NyaaPantsu/nyaa/util/log"
	"time"

	"github.com/NyaaPantsu/nyaa/config"
)

const (
	// UserStatusBanned : Int for User status banned
	UserStatusBanned = -1
	// UserStatusMember : Int for User status member
	UserStatusMember = 0
	// UserStatusTrusted : Int for User status trusted
	UserStatusTrusted = 1
	// UserStatusModerator : Int for User status moderator
	UserStatusModerator = 2
	// UserStatusScraped : Int for User status scrapped
	UserStatusScraped = 3
)

// User model
type User struct {
	ID             uint      `gorm:"column:user_id;primary_key"`
	Username       string    `gorm:"column:username;unique"`
	Password       string    `gorm:"column:password"`
	Email          string    `gorm:"column:email;unique"`
	Status         int       `gorm:"column:status"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
	APIToken       string    `gorm:"column:api_token"`
	APITokenExpiry time.Time `gorm:"column:api_token_expiry"`
	Language       string    `gorm:"column:language"`
	Theme          string    `gorm:"column:theme"`
	Mascot         string    `gorm:"column:mascot"`
	MascotURL      string    `gorm:"column:mascot_url"`
	UserSettings   string    `gorm:"column:settings"`

	// TODO: move this to PublicUser
	Followers []User // Don't work `gorm:"foreignkey:user_id;associationforeignkey:follower_id;many2many:user_follows"`
	Likings   []User // Don't work `gorm:"foreignkey:follower_id;associationforeignkey:user_id;many2many:user_follows"`

	MD5           string         `json:"md5" gorm:"column:md5"` // Hash of email address, used for Gravatar
	Torrents      []Torrent      `gorm:"ForeignKey:UploaderID"`
	Notifications []Notification `gorm:"ForeignKey:UserID"`

	UnreadNotifications int          `gorm:"-"` // We don't want to loop every notifications when accessing user unread notif
	Settings            UserSettings `gorm:"-"` // We don't want to load settings everytime, stock it as a string, parse it when needed
}

// UserJSON : User model conversion in JSON
type UserJSON struct {
	ID          uint   `json:"user_id"`
	Username    string `json:"username"`
	Status      int    `json:"status"`
	APIToken    string `json:"token"`
	MD5         string `json:"md5"`
	CreatedAt   string `json:"created_at"`
	LikingCount int    `json:"liking_count"`
	LikedCount  int    `json:"liked_count"`
}

// UserFollows association table : different users following eachother
type UserFollows struct {
	UserID     uint `gorm:"column:user_id"`
	FollowerID uint `gorm:"column:following"`
}

// UserUploadsOld model : Is it deprecated?
type UserUploadsOld struct {
	Username  string `gorm:"column:username"`
	TorrentID uint   `gorm:"column:torrent_id"`
}

// UserSettings : Struct for user settings, not a model
type UserSettings struct {
	Settings map[string]bool `json:"settings"`
}

/*
 * User Model
 */

// Size : Returns the total size of memory recursively allocated for this struct
func (u User) Size() (s int) {
	s += 4 + // ints
		6*2 + // string pointers
		4*3 + //time.Time
		3*2 + // arrays
		// string arrays
		len(u.Username) + len(u.Password) + len(u.Email) + len(u.APIToken) + len(u.MD5) + len(u.Language) + len(u.Theme)
	s *= 8

	// Ignoring foreign key users. Fuck them.

	return
}

// IsBanned : Return true if user is banned
func (u *User) IsBanned() bool {
	return u.Status == UserStatusBanned
}

// IsMember : Return true if user is member
func (u *User) IsMember() bool {
	return u.Status == UserStatusMember
}

// IsTrusted : Return true if user is tusted
func (u *User) IsTrusted() bool {
	return u.Status == UserStatusTrusted
}

// IsModerator : Return true if user is moderator
func (u *User) IsModerator() bool {
	return u.Status == UserStatusModerator
}

// IsScraped : Return true if user is a scrapped user
func (u *User) IsScraped() bool {
	return u.Status == UserStatusScraped
}

// GetUnreadNotifications : Get unread notifications from a user
func (u *User) GetUnreadNotifications() int {
	if u.UnreadNotifications == 0 {
		for _, notif := range u.Notifications {
			if !notif.Read {
				u.UnreadNotifications++
			}
		}
	}
	return u.UnreadNotifications
}

// HasAdmin checks that user has an admin permission. Deprecated
func (u *User) HasAdmin() bool {
	return u.IsModerator()
}

// CurrentOrAdmin check that user has admin permission or user is the current user.
func (u *User) CurrentOrAdmin(userID uint) bool {
	log.Debugf("user.ID == userID %d %d %s", u.ID, userID, u.ID == userID)
	return (u.IsModerator() || u.ID == userID)
}

// CurrentUserIdentical check that userID is same as current user's ID.
// TODO: Inline this (won't go do this for us?)
func (u *User) CurrentUserIdentical(userID uint) bool {
	return u.ID == userID
}

// NeedsCaptcha : Check if a user needs captcha
func (u *User) NeedsCaptcha() bool {
	// Trusted members & Moderators don't
	return !(u.IsTrusted() || u.IsModerator())
}

// GetRole : Get the status/role of a user
func (u *User) GetRole() string {
	switch u.Status {
	case UserStatusBanned:
		return "Banned"
	case UserStatusMember:
		return "Member"
	case UserStatusScraped:
		return "Member"
	case UserStatusTrusted:
		return "Trusted Member"
	case UserStatusModerator:
		return "Moderator"
	}
	return "Member"
}

// IsFollower : Check if a user is following another
func (follower *User) IsFollower(u *User) bool {
	var likingUserCount int
	ORM.Model(&UserFollows{}).Where("user_id = ? and following = ?", follower.ID, u.ID).Count(&likingUserCount)
	return likingUserCount != 0
}

// ToJSON : Conversion of a user model to json
func (u *User) ToJSON() UserJSON {
	json := UserJSON{
		ID:          u.ID,
		Username:    u.Username,
		APIToken:    u.APIToken,
		MD5:         u.MD5,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		LikingCount: len(u.Followers),
		LikedCount:  len(u.Likings),
	}
	return json
}

// GetLikings : Gets who is followed by the user
func (u *User) GetLikings() {
	var liked []User
	ORM.Joins("JOIN user_follows on user_follows.following=?", u.ID).Where("users.user_id = user_follows.user_id").Group("users.user_id").Find(&liked)
	u.Likings = liked
}

// GetFollowers : Gets who is following the user
func (u *User) GetFollowers() {
	var likings []User
	ORM.Joins("JOIN user_follows on user_follows.user_id=?", u.ID).Where("users.user_id = user_follows.following").Group("users.user_id").Find(&likings)
	u.Followers = likings
}

// SetFollow : Makes a user follow another
func (u *User) SetFollow(follower *User) {
	if follower.ID > 0 && u.ID > 0 {
		var userFollows = UserFollows{UserID: u.ID, FollowerID: follower.ID}
		ORM.Create(&userFollows)
	}
}

// RemoveFollow : Remove a user following another
func (u *User) RemoveFollow(follower *User) {
	if follower.ID > 0 && u.ID > 0 {
		var userFollows = UserFollows{UserID: u.ID, FollowerID: follower.ID}
		ORM.Delete(&userFollows)
	}
}

/*
 * Old User
 */

// TableName : Return the name of OldComment table
func (c UserUploadsOld) TableName() string {
	// is this needed here?
	return config.Conf.Models.UploadsOldTableName
}

/*
 * User Settings
 */

// Get a user setting by keyname
func (s *UserSettings) Get(key string) bool {
	if val, ok := s.Settings[key]; ok {
		return val
	}
	return config.Conf.Users.DefaultUserSettings[key]
}

// GetSettings : get all user settings
func (s *UserSettings) GetSettings() map[string]bool {
	return s.Settings
}

// Set a user setting by keyname
func (s *UserSettings) Set(key string, val bool) {
	if s.Settings == nil {
		s.Settings = make(map[string]bool)
	}
	s.Settings[key] = val
}

// ToDefault : Set user settings to default
func (s *UserSettings) ToDefault() {
	s.Settings = config.Conf.Users.DefaultUserSettings
}

func (s *UserSettings) initialize() {
	s.Settings = make(map[string]bool)
}

// SaveSettings : Format settings into a json string for preparing before user insertion
func (u *User) SaveSettings() {
	byteArray, err := json.Marshal(u.Settings)

	if err != nil {
		fmt.Print(err)
	}
	u.UserSettings = string(byteArray)
}

// ParseSettings : Function to parse json string into usersettings struct, only parse if necessary
func (u *User) ParseSettings() {
	if len(u.Settings.GetSettings()) == 0 && u.UserSettings != "" {
		u.Settings.initialize()
		json.Unmarshal([]byte(u.UserSettings), &u.Settings)
	} else if len(u.Settings.GetSettings()) == 0 && u.UserSettings != "" {
		u.Settings.initialize()
		u.Settings.ToDefault()
	}
}
