package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"h-ui/model/dto"
	"h-ui/model/entity"
	"h-ui/model/vo"
	"h-ui/service"
	"h-ui/util"
)

func Login(c *gin.Context) {
	loginDto, err := validateField(c, dto.LoginDto{})
	if err != nil {
		return
	}
	token, err := service.Login(*loginDto.Username, *loginDto.Pass)
	if err != nil {
		vo.Fail(err.Error(), c)
		return
	}
	vo.Success(token, c)
}

func PageAccount(c *gin.Context) {
	accountPageDto, err := validateField(c, dto.AccountPageDto{})
	if err != nil {
		return
	}
	accounts, total, err := service.PageAccount(accountPageDto)
	if err != nil {
		vo.Fail(err.Error(), c)
		return
	}

	var accountVos []vo.AccountVo
	for _, item := range accounts {
		accountVo := vo.AccountVo{
			Username:   *item.Username,
			Quota:      *item.Quota,
			Download:   *item.Download,
			Upload:     *item.Upload,
			ExpireTime: *item.ExpireTime,
			Deleted:    *item.Deleted,
			BaseVo: vo.BaseVo{
				Id:         *item.Id,
				CreateTime: *item.CreateTime,
			},
		}
		accountVos = append(accountVos, accountVo)
	}
	accountPageVo := vo.AccountPageVo{
		AccountVos: accountVos,
		Total:      total,
	}
	vo.Success(accountPageVo, c)
}

func SaveAccount(c *gin.Context) {
	accountSaveDto, err := validateField(c, dto.AccountSaveDto{})
	if err != nil {
		return
	}

	if service.ExistAccountUsername(*accountSaveDto.Username) {
		vo.Fail(fmt.Sprintf("用户名%s已存在", *accountSaveDto.Username), c)
		return
	}

	passEncrypt := util.SHA224String(*accountSaveDto.Pass)
	conPassEncrypt := util.SHA224String(fmt.Sprintf("%s@%s", *accountSaveDto.Username, *accountSaveDto.ConPass))
	account := entity.Account{
		Username:   accountSaveDto.Username,
		Pass:       &passEncrypt,
		ConPass:    &conPassEncrypt,
		Quota:      accountSaveDto.Quota,
		ExpireTime: accountSaveDto.ExpireTime,
		Deleted:    accountSaveDto.Deleted,
	}
	err = service.SaveAccount(account)
	if err != nil {
		vo.Fail(err.Error(), c)
		return
	}
	vo.Success(nil, c)
}

func DeleteAccount(c *gin.Context) {
	idDto, err := validateField(c, dto.IdDto{})
	if err != nil {
		return
	}
	err = service.DeleteAccount([]int64{*idDto.Id})
	if err != nil {
		vo.Fail(err.Error(), c)
		return
	}
	vo.Success(nil, c)
}

func UpdateAccount(c *gin.Context) {
	accountUpdateDto, err := validateField(c, dto.AccountUpdateDto{})
	if err != nil {
		return
	}
	account := entity.Account{
		Username:   accountUpdateDto.Username,
		Pass:       accountUpdateDto.Pass,
		ConPass:    accountUpdateDto.ConPass,
		Quota:      accountUpdateDto.Quota,
		ExpireTime: accountUpdateDto.ExpireTime,
		Deleted:    accountUpdateDto.Deleted,
		BaseEntity: entity.BaseEntity{
			Id: accountUpdateDto.Id,
		},
	}
	if err = service.UpdateAccount(account); err != nil {
		vo.Fail(err.Error(), c)
		return
	}
	vo.Success(nil, c)
}
