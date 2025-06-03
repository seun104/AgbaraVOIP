package services

import (
	"context"
	"time"
	"github.com/user/agbaravoip_golang/internal/domain"
)

type ConferenceService interface {
	GetConferenceBySID(ctx context.Context, sid string) (*domain.Conference, error)
	GetConferenceByName(ctx context.Context, accountSid, name string) (*domain.Conference, error)
	CreateConference(ctx context.Context, accountSid, name, sid string) (*domain.Conference, error)
	GetOrCreateConference(ctx context.Context, accountSid, name string) (*domain.Conference, error)
	UpdateConferenceStatus(ctx context.Context, sid string, status domain.ConferenceStatus) error
	EndConference(ctx context.Context, sid string, endTime time.Time) error

	AddParticipant(ctx context.Context, confSid, callSid, pSid, accountSid string, isMuted, isModerator bool) (*domain.ConferenceParticipant, error)
	GetParticipant(ctx context.Context, pSid string) (*domain.ConferenceParticipant, error)
	GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error)
	UpdateParticipantMuteStatus(ctx context.Context, pSid string, isMuted bool) error
	UpdateParticipantModeratorStatus(ctx context.Context, pSid string, isModerator bool) error
	RemoveParticipant(ctx context.Context, pSid string, leaveTime time.Time) error
	ListParticipants(ctx context.Context, confSid string) ([]*domain.ConferenceParticipant, error)
}
