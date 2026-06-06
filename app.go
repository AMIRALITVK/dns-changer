package main

import (
	"context"
	"math/rand"

	"dns-changer/dns"
	"dns-changer/pinger"
	"dns-changer/profiles"
)

type App struct {
	ctx    context.Context
	dnsMgr dns.Manager
	store  *profiles.Store
}

func NewApp() *App {
	mgr, _ := dns.NewManager()
	store, _ := profiles.NewStore()
	return &App{
		dnsMgr: mgr,
		store:  store,
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetProfiles() []profiles.Profile {
	if a.store == nil {
		return []profiles.Profile{}
	}
	list, err := a.store.Load()
	if err != nil || list == nil {
		return []profiles.Profile{}
	}
	return list
}

func (a *App) AddProfile(name string, servers []string) error {
	if a.store == nil {
		return nil
	}
	list, err := a.store.Load()
	if err != nil {
		list = []profiles.Profile{}
	}

	id := generateID()
	list = append(list, profiles.Profile{
		ID:      id,
		Name:    name,
		Servers: servers,
	})
	return a.store.Save(list)
}

func (a *App) UpdateProfile(id string, name string, servers []string) error {
	if a.store == nil {
		return nil
	}
	list, err := a.store.Load()
	if err != nil {
		return err
	}

	for i, p := range list {
		if p.ID == id {
			list[i].Name = name
			list[i].Servers = servers
			return a.store.Save(list)
		}
	}
	return nil
}

func (a *App) DeleteProfile(id string) error {
	if a.store == nil {
		return nil
	}
	list, err := a.store.Load()
	if err != nil {
		return err
	}

	var newList []profiles.Profile
	for _, p := range list {
		if p.ID != id {
			newList = append(newList, p)
		}
	}
	return a.store.Save(newList)
}

func (a *App) ReorderProfiles(ids []string) error {
	if a.store == nil {
		return nil
	}
	list, err := a.store.Load()
	if err != nil {
		return err
	}
	profileMap := make(map[string]profiles.Profile, len(list))
	for _, p := range list {
		profileMap[p.ID] = p
	}
	newList := make([]profiles.Profile, 0, len(list))
	for _, id := range ids {
		if p, ok := profileMap[id]; ok {
			newList = append(newList, p)
		}
	}
	return a.store.Save(newList)
}

func (a *App) SetDNS(servers []string) error {
	if a.dnsMgr == nil {
		return nil
	}
	return a.dnsMgr.SetDNS(servers)
}

func (a *App) RemoveDNS() error {
	if a.dnsMgr == nil {
		return nil
	}
	return a.dnsMgr.RemoveDNS()
}

func (a *App) GetCurrentDNS() []string {
	if a.dnsMgr == nil {
		return []string{}
	}
	servers, err := a.dnsMgr.GetCurrentDNS()
	if err != nil || servers == nil {
		return []string{}
	}
	return servers
}

func (a *App) PingServer(server string) pinger.Result {
	return pinger.Ping(server)
}

func (a *App) GetPlatform() string {
	return dns.GetPlatform()
}

func (a *App) GetActiveInterface() string {
	if a.dnsMgr == nil {
		return ""
	}
	iface, err := a.dnsMgr.GetActiveInterface()
	if err != nil {
		return ""
	}
	return iface
}

func generateID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
