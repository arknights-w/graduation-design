package engine

import (
	"Airfone/api/errorpb"
	"sync"
	"time"
)

type ServiceMap struct {
	sync.RWMutex
	services map[int32]*Service
}

func NewServiceMap() *ServiceMap {
	return &ServiceMap{
		services: make(map[int32]*Service),
	}
}

func (sm *ServiceMap) Add(now int64, id int32, s *Service) (*Service, error) {
	sm.Lock()
	defer sm.Unlock()
	if _, ok := sm.services[id]; ok {
		return nil, errorpb.ErrorInsertAlreadyExist("this service is already existed")
	}
	if s.ID == 0 {
		s.ID = id
	}
	s.keepalive = now
	sm.services[id] = s
	return s, nil
}

func (sm *ServiceMap) Update(now int64, id int32, s *Service) (*Service, error) {
	sm.Lock()
	defer sm.Unlock()
	var (
		service *Service
		ok      bool
	)
	if service, ok = sm.services[id]; !ok {
		return nil, errorpb.ErrorUpdateInvalid("this service is not exist")
	}
	if s.IP != "" {
		service.IP = s.IP
	}
	if s.Port != 0 {
		service.Port = s.Port
	}
	if s.Rely != nil {
		service.Rely = s.Rely
	}
	if s.Schema != nil {
		service.Schema = s.Schema
	}
	service.keepalive = now
	return service, nil
}

func (sm *ServiceMap) Get(id int32) (*Service, error) {
	sm.RLock()
	defer sm.RUnlock()
	var (
		service *Service
		ok      bool
	)
	if service, ok = sm.services[id]; !ok {
		return nil, errorpb.ErrorSearchInvalid("this service is not exist")
	}
	return service, nil
}

func (sm *ServiceMap) Delete(now int64, id int32) (*Service, error) {
	sm.Lock()
	defer sm.Unlock()
	var (
		service *Service
		ok      bool
	)
	if service, ok = sm.services[id]; !ok {
		return nil, errorpb.ErrorDeleteInvalid("this service is not exist")
	}
	service.keepalive = now
	service.Status = DROPPING
	return service, nil
}

// 依照时间将所有过期的数据删除，并返回被删除的数据
func (sm *ServiceMap) BatchDeleteByCheckTime(now int64, duration time.Duration) ([]*Service, error) {
	var (
		servs = make([]*Service, 0, 3)
		limit = now - int64(duration)
	)
	sm.Lock()
	defer sm.Unlock()
	for _, s := range sm.services {
		if s.keepalive < limit {
			servs = append(servs, s)
		}
	}
	for _, s := range servs {
		delete(sm.services, s.ID)
	}
	return servs, nil
}

func (sm *ServiceMap) BatchAdd(now int64, servs []*Service) error {
	sm.Lock()
	defer sm.Unlock()
	for _, s := range servs {
		if _, ok := sm.services[s.ID]; ok {
			return errorpb.ErrorInsertAlreadyExist("this service is already existed")
		}
		sm.services[s.ID] = s
	}
	return nil
}
