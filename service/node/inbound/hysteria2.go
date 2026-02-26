package inbound

import (
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/hysteria2"
	CM "github.com/sagernet/sing-box/service/manager/constant"
	"github.com/sagernet/sing-box/service/node/constant"
)

type Hysteria2Manager struct {
	access   sync.Mutex
	inbounds map[string]*Hysteria2UserManager
}

func NewHysteria2Manager() *Hysteria2Manager {
	return &Hysteria2Manager{
		inbounds: make(map[string]*Hysteria2UserManager),
	}
}

func (m *Hysteria2Manager) AddUserManager(inbound adapter.Inbound) error {
	m.access.Lock()
	defer m.access.Unlock()
	m.inbounds[inbound.Tag()] = &Hysteria2UserManager{
		inbound:  inbound.(*hysteria2.Inbound),
		usersMap: make(map[string]option.Hysteria2User),
	}
	return nil
}

func (m *Hysteria2Manager) GetUserManager(tag string) (constant.UserManager, bool) {
	m.access.Lock()
	defer m.access.Unlock()
	inbound, ok := m.inbounds[tag]
	return inbound, ok
}

func (m *Hysteria2Manager) GetUserManagerTags() []string {
	m.access.Lock()
	defer m.access.Unlock()
	tags := make([]string, 0, len(m.inbounds))
	for tag, _ := range m.inbounds {
		tags = append(tags, tag)
	}
	return tags
}

type Hysteria2UserManager struct {
	inbound  *hysteria2.Inbound
	usersMap map[string]option.Hysteria2User

	mtx sync.Mutex
}

func (i *Hysteria2UserManager) postUpdate() {
	users := make([]option.Hysteria2User, 0, len(i.usersMap))
	for _, user := range i.usersMap {
		users = append(users, user)
	}
	i.inbound.UpdateUsers(users)
}

func (i *Hysteria2UserManager) UpdateUser(user CM.User) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	i.usersMap[user.Username] = option.Hysteria2User{Name: user.Username, Password: user.Password}
	i.postUpdate()
}

func (i *Hysteria2UserManager) UpdateUsers(users []CM.User) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	clear(i.usersMap)
	for _, user := range users {
		i.usersMap[user.Username] = option.Hysteria2User{Name: user.Username, Password: user.Password}
	}
	i.postUpdate()
}

func (i *Hysteria2UserManager) DeleteUser(username string) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	delete(i.usersMap, username)
	i.postUpdate()
}
