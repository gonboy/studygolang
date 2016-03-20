// Copyright 2016 The StudyGolang Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// http://studygolang.com
// Author：polaris	polaris@studygolang.com

package logic

import (
	"errors"
	"math/rand"
	"model"
	"net/url"
	"time"

	"github.com/polaris1119/goutils"
	"github.com/polaris1119/logger"
	"golang.org/x/net/context"

	. "db"
)

var DefaultAvatars = []string{
	"gopher_aqua.jpg", "gopher_boy.jpg", "gopher_brown.jpg", "gopher_gentlemen.jpg",
	"gopher_strawberry.jpg", "gopher_strawberry_bg.jpg", "gopher_teal.jpg",
	"gopher01.png", "gopher02.png", "gopher03.png", "gopher04.png",
	"gopher05.png", "gopher06.png", "gopher07.png", "gopher08.png",
	"gopher09.png", "gopher10.png", "gopher11.png", "gopher12.png",
	"gopher13.png", "gopher14.png", "gopher15.png", "gopher16.png",
	"gopher17.png", "gopher18.png", "gopher19.png", "gopher20.png",
	"gopher21.png", "gopher22.png", "gopher23.png", "gopher24.png",
	"gopher25.png", "gopher26.png", "gopher27.png", "gopher28.png",
}

type UserLogic struct{}

var DefaultUser = UserLogic{}

// CreateUser 创建用户
func (self UserLogic) CreateUser(ctx context.Context, form url.Values) (errMsg string, err error) {
	objLog := GetLogger(ctx)

	if self.UserExists(ctx, "email", form.Get("email")) {
		err = errors.New("该邮箱已注册过")
		return
	}
	if self.UserExists(ctx, "username", form.Get("username")) {
		err = errors.New("用户名已存在")
		return
	}

	user := &model.User{}
	err = schemaDecoder.Decode(user, form)
	if err != nil {
		objLog.Errorln("user schema Decode error:", err)
		errMsg = err.Error()
		return
	}

	// 随机给一个默认头像
	user.Avatar = DefaultAvatars[rand.Intn(len(DefaultAvatars))]
	_, err = MasterDB.Insert(user)
	if err != nil {
		errMsg = "内部服务器错误"
		objLog.Errorln(errMsg, ":", err)
		return
	}

	// 存用户登录信息
	userLogin := &model.UserLogin{}
	err = schemaDecoder.Decode(userLogin, form)
	if err != nil {
		errMsg = err.Error()
		objLog.Errorln("CreateUser error:", err)
		return
	}
	userLogin.Uid = user.Uid
	if _, err = MasterDB.Insert(userLogin); err != nil {
		errMsg = "内部服务器错误"
		logger.Errorln(errMsg, ":", err)
		return
	}

	// 存用户角色信息
	userRole := &model.UserRole{}
	// 默认为初级会员
	userRole.Roleid = Roles[len(Roles)-1].Roleid
	userRole.Uid = user.Uid
	if _, err = MasterDB.Insert(userRole); err != nil {
		objLog.Errorln("userRole insert Error:", err)
	}

	// 存用户活跃信息，初始活跃+2
	userActive := &model.UserActive{}
	userActive.Uid = user.Uid
	userActive.Username = user.Username
	userActive.Avatar = user.Avatar
	userActive.Email = user.Email
	userActive.Weight = 2
	if _, err = MasterDB.Insert(userActive); err != nil {
		objLog.Errorln("UserActive insert Error:", err)
	}
	return
}

// Update 更新用户信息
func (self UserLogic) Update(ctx context.Context, uid int, form url.Values) (errMsg string, err error) {
	objLog := GetLogger(ctx)

	if form.Get("open") != "1" {
		form.Set("open", "0")
	}

	user := &model.User{}
	err = schemaDecoder.Decode(user, form)
	if err != nil {
		objLog.Errorln("userlogic update, schema decode error:", err)
		errMsg = "服务内部错误"
		return
	}

	cols := "name,open,city,company,github,weibo,website,monlog,introduce"
	_, err = MasterDB.Id(uid).Cols(cols).Update(user)
	if err != nil {
		objLog.Errorf("更新用户 【%s】 信息失败：%s", uid, err)
		errMsg = "对不起，服务器内部错误，请稍后再试！"
		return
	}

	// 修改用户资料，活跃度+1
	go self.IncrUserWeight("uid", uid, 1)

	return
}

// ChangeAvatar 更换头像
func (UserLogic) ChangeAvatar(ctx context.Context, uid int, avatar string) (err error) {
	changeData := map[string]interface{}{"avatar": avatar}
	_, err = MasterDB.Table(new(model.User)).Id(uid).Update(changeData)
	if err == nil {
		_, err = MasterDB.Table(new(model.UserActive)).Id(uid).Update(changeData)
	}

	return
}

// UserExists 判断用户是否存在
func (UserLogic) UserExists(ctx context.Context, field, val string) bool {
	objLog := GetLogger(ctx)

	userLogin := &model.UserLogin{}
	_, err := MasterDB.Where(field+"=?", val).Get(userLogin)
	if err != nil || userLogin.Uid == 0 {
		if err != nil {
			objLog.Errorln("user logic UserExists error:", err)
		}
		return false
	}
	return true
}

// EmailOrUsernameExists 判断指定的邮箱（email）或用户名是否存在
func (UserLogic) EmailOrUsernameExists(ctx context.Context, email, username string) bool {
	objLog := GetLogger(ctx)

	userLogin := &model.UserLogin{}
	_, err := MasterDB.Where("email=?", email).Or("username=?", username).Get(userLogin)
	if err != nil || userLogin.Uid != 0 {
		if err != nil {
			objLog.Errorln("user logic EmailOrUsernameExists error:", err)
		}
		return false
	}
	return true
}

func (self UserLogic) FindUserInfos(ctx context.Context, uids []int) map[int]*model.User {
	objLog := GetLogger(ctx)

	var usersMap = make(map[int]*model.User)
	if err := MasterDB.In("uid", uids).Find(&usersMap); err != nil {
		objLog.Infoln("user logic FindAll not record found:")
		return nil
	}

	// usersMap := make(map[int]*model.User, len(users))
	// for _, user := range users {
	// 	if user == nil || user.Uid == 0 {
	// 		continue
	// 	}
	// 	usersMap[user.Uid] = user
	// }
	return usersMap
}

func (self UserLogic) FindOne(ctx context.Context, field string, val interface{}) *model.User {
	objLog := GetLogger(ctx)

	user := &model.User{}
	_, err := MasterDB.Where(field+"=?", val).Get(user)
	if err != nil {
		objLog.Errorln("user logic FindOne error:", err)
	}
	return user
}

var (
	ErrUsername = errors.New("用户名不存在")
	ErrPasswd   = errors.New("密码错误")
)

// Login 登录；成功返回用户登录信息(user_login)
func (self UserLogic) Login(ctx context.Context, username, passwd string) (*model.UserLogin, error) {
	objLog := GetLogger(ctx)

	userLogin := &model.UserLogin{}
	_, err := MasterDB.Where("username=? OR email=?", username, username).Get(userLogin)
	if err != nil {
		objLog.Errorf("user %q login failure: %s", username, err)
		return nil, errors.New("内部错误，请稍后再试！")
	}
	// 校验用户
	if userLogin.Uid == 0 {
		objLog.Infof("user %q is not exists!", username)
		return nil, ErrUsername
	}

	// 检验用户是否审核通过，暂时只有审核通过的才能登录
	user := &model.User{}
	MasterDB.Id(userLogin.Uid).Get(user)
	if user.Status != model.UserStatusAudit {
		objLog.Infof("用户 %q 的状态非审核通过, 用户的状态值：%d", username, user.Status)
		var errMap = map[int]error{
			model.UserStatusNoAudit: errors.New("您的账号未激活，请到注册邮件中进行激活操作！"),
			model.UserStatusRefuse:  errors.New("您的账号审核拒绝"),
			model.UserStatusFreeze:  errors.New("您的账号因为非法发布信息已被冻结，请联系管理员！"),
			model.UserStatusOutage:  errors.New("您的账号因为非法发布信息已被停号，请联系管理员！"),
		}
		return nil, errMap[user.Status]
	}

	md5Passwd := goutils.Md5(passwd + userLogin.Passcode)
	objLog.Debugf("passwd: %s, passcode: %s, md5passwd: %s, dbpasswd: %s", passwd, userLogin.Passcode, md5Passwd, userLogin.Passwd)
	if md5Passwd != userLogin.Passwd {
		objLog.Infof("用户名 %q 填写的密码错误", username)
		return nil, ErrPasswd
	}

	go func() {
		self.IncrUserWeight("uid", userLogin.Uid, 1)
		self.RecordLoginTime(username)
	}()

	return userLogin, nil
}

// UpdatePasswd 更新用户密码
func (self UserLogic) UpdatePasswd(ctx context.Context, username, curPasswd, newPasswd string) (string, error) {
	_, err := self.Login(ctx, username, curPasswd)
	if err != nil {
		return "原密码填写错误", err
	}

	userLogin := &model.UserLogin{}
	newPasswd = userLogin.GenMd5Passwd(newPasswd)

	changeData := map[string]interface{}{
		"passwd":   newPasswd,
		"passcode": userLogin.Passcode,
	}
	_, err = MasterDB.Table(userLogin).Where("username=?", username).Update(changeData)
	if err != nil {
		logger.Errorf("用户 %s 更新密码错误：%s", username, err)
		return "对不起，内部服务错误！", err
	}
	return "", nil
}

func (self UserLogic) ResetPasswd(ctx context.Context, email, passwd string) (string, error) {
	objLog := GetLogger(ctx)

	userLogin := &model.UserLogin{}
	passwd = userLogin.GenMd5Passwd(passwd)

	changeData := map[string]interface{}{
		"passwd":   passwd,
		"passcode": userLogin.Passcode,
	}
	_, err := MasterDB.Table(userLogin).Where("email=?", email).Update(changeData)
	if err != nil {
		objLog.Errorf("用户 %s 更新密码错误：%s", email, err)
		return "对不起，内部服务错误！", err
	}
	return "", nil
}

// Activate 用户激活
func (self UserLogic) Activate(ctx context.Context, email, uuid string, timestamp int64, sign string) (*model.User, error) {
	objLog := GetLogger(ctx)

	realSign := DefaultEmail.genActivateSign(email, uuid, timestamp)
	if sign != realSign {
		return nil, errors.New("签名非法！")
	}

	user := self.FindOne(ctx, "email", email)
	if user.Uid == 0 {
		return nil, errors.New("邮箱非法")
	}

	user.Status = model.UserStatusAudit

	_, err := MasterDB.Id(user.Uid).Update(user)
	if err != nil {
		objLog.Errorf("activate [%s] failure:%s", email, err)
		return nil, err
	}

	return user, nil
}

// 增加或减少用户活跃度
func (UserLogic) IncrUserWeight(field string, value interface{}, weight int) {
	_, err := MasterDB.Where(field+"=?", value).Incr("weight", weight).Update(new(model.UserActive))
	if err != nil {
		logger.Errorln("UserActive update Error:", err)
	}
}

// RecordLoginTime 记录用户最后登录时间
func (UserLogic) RecordLoginTime(username string) error {
	_, err := MasterDB.Table(new(model.UserLogin)).Where("username=?", username).
		Update(map[string]interface{}{"login_time": time.Now()})
	if err != nil {
		logger.Errorf("记录用户 %q 登录时间错误：%s", username, err)
	}
	return err
}
