package deploy

type ChannelDeploys struct {
	store Store
}

func NewChannelDeploys(store Store) *ChannelDeploys {
	return &ChannelDeploys{store: store}
}

func (repo *ChannelDeploys) Current(channelID string) (Deploy, bool) {
	d, ok := repo.store.Get(channelID)
	return d, ok && d.FinishedAt.IsZero()
}

func (repo *ChannelDeploys) Start(channelID string, d Deploy) (Deploy, bool) {
	for {
		current, ok := repo.Current(channelID)
		if !ok {
			break
		}

		if current.User.ID != d.User.ID {
			return current, false
		}

		repo.Finish(channelID)
	}

	d.Start()
	repo.store.Set(channelID, d)

	return d, true
}

func (repo *ChannelDeploys) Finish(channelID string) (Deploy, bool) {
	current, ok := repo.Current(channelID)
	if !ok {
		return current, false
	}

	current.Finish()
	repo.store.Set(channelID, current)

	return current, true
}

func (repo *ChannelDeploys) Abort(channelID, reason string) (Deploy, bool) {
	current, ok := repo.Current(channelID)
	if !ok {
		return current, false
	}

	current.Abort(reason)
	repo.store.Set(channelID, current)

	return current, true
}
