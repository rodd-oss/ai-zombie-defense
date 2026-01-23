package db

import (
	internal "ai-zombie-defense/backend-api/db/internal/db"
)

type DBTX = internal.DBTX
type Queries = internal.Queries

func New() *Queries {
	return internal.New()
}

// Aliases for all generated types
type GetPrestigeCosmeticsParams = internal.GetPrestigeCosmeticsParams
type GrantCosmeticToPlayerParams = internal.GrantCosmeticToPlayerParams
type CreateCurrencyTransactionParams = internal.CreateCurrencyTransactionParams
type GetCurrencyTransactionsByPlayerParams = internal.GetCurrencyTransactionsByPlayerParams
type GetCurrencyTransactionsByPlayerAndTypeParams = internal.GetCurrencyTransactionsByPlayerAndTypeParams
type AcceptFriendRequestParams = internal.AcceptFriendRequestParams
type CreateFriendRequestParams = internal.CreateFriendRequestParams
type DeclineFriendRequestParams = internal.DeclineFriendRequestParams
type GetFriendRequestParams = internal.GetFriendRequestParams
type ListFriendsRow = internal.ListFriendsRow
type ListPendingIncomingRow = internal.ListPendingIncomingRow
type ListPendingOutgoingRow = internal.ListPendingOutgoingRow
type CreateJoinTokenParams = internal.CreateJoinTokenParams
type GetAllTimeLeaderboardRow = internal.GetAllTimeLeaderboardRow
type GetDailyLeaderboardRow = internal.GetDailyLeaderboardRow
type GetWeeklyLeaderboardRow = internal.GetWeeklyLeaderboardRow
type CreateLoadoutParams = internal.CreateLoadoutParams
type DeleteLoadoutCosmeticBySlotParams = internal.DeleteLoadoutCosmeticBySlotParams
type GetLoadoutCosmeticBySlotParams = internal.GetLoadoutCosmeticBySlotParams
type GetLoadoutCosmeticsRow = internal.GetLoadoutCosmeticsRow
type InsertLoadoutCosmeticParams = internal.InsertLoadoutCosmeticParams
type UpdateLoadoutActiveParams = internal.UpdateLoadoutActiveParams
type CreateLootTableEntryParams = internal.CreateLootTableEntryParams
type GetLootTableEntriesWithCosmeticDetailsRow = internal.GetLootTableEntriesWithCosmeticDetailsRow
type UpdateLootTableEntryParams = internal.UpdateLootTableEntryParams
type CreateLootTableParams = internal.CreateLootTableParams
type UpdateLootTableParams = internal.UpdateLootTableParams
type CreateMatchParams = internal.CreateMatchParams
type GetPlayerMatchHistoryParams = internal.GetPlayerMatchHistoryParams
type GetPlayerMatchHistoryRow = internal.GetPlayerMatchHistoryRow
type UpdateMatchOutcomeParams = internal.UpdateMatchOutcomeParams
type CosmeticItem = internal.CosmeticItem
type CurrencyTransaction = internal.CurrencyTransaction
type Friend = internal.Friend
type JoinToken = internal.JoinToken
type LeaderboardEntry = internal.LeaderboardEntry
type Loadout = internal.Loadout
type LoadoutCosmetic = internal.LoadoutCosmetic
type LootTable = internal.LootTable
type LootTableEntry = internal.LootTableEntry
type Match = internal.Match
type Player = internal.Player
type PlayerCosmetic = internal.PlayerCosmetic
type PlayerMatchStat = internal.PlayerMatchStat
type PlayerProgression = internal.PlayerProgression
type PlayerSetting = internal.PlayerSetting
type Server = internal.Server
type ServerFavorite = internal.ServerFavorite
type Session = internal.Session
type CreatePlayerParams = internal.CreatePlayerParams
type UpdatePlayerLastLoginParams = internal.UpdatePlayerLastLoginParams
type UpdatePlayerPasswordParams = internal.UpdatePlayerPasswordParams
type UpdatePlayerProfileParams = internal.UpdatePlayerProfileParams
type GetPlayerCosmeticParams = internal.GetPlayerCosmeticParams
type GetPlayerCosmeticRow = internal.GetPlayerCosmeticRow
type GetPlayerCosmeticsRow = internal.GetPlayerCosmeticsRow
type CreatePlayerMatchStatsParams = internal.CreatePlayerMatchStatsParams
type GetPlayerMatchStatsParams = internal.GetPlayerMatchStatsParams
type AddDataCurrencyParams = internal.AddDataCurrencyParams
type IncrementExperienceParams = internal.IncrementExperienceParams
type IncrementMatchStatsParams = internal.IncrementMatchStatsParams
type SetDataCurrencyParams = internal.SetDataCurrencyParams
type UpdateLevelParams = internal.UpdateLevelParams
type UpdatePlayerProgressionParams = internal.UpdatePlayerProgressionParams
type UpsertPlayerSettingsParams = internal.UpsertPlayerSettingsParams
type AddFavoriteParams = internal.AddFavoriteParams
type GetFavoriteParams = internal.GetFavoriteParams
type ListPlayerFavoritesRow = internal.ListPlayerFavoritesRow
type RemoveFavoriteParams = internal.RemoveFavoriteParams
type CreateServerParams = internal.CreateServerParams
type ListActiveServersParams = internal.ListActiveServersParams
type UpdateServerHeartbeatParams = internal.UpdateServerHeartbeatParams
type CreateSessionParams = internal.CreateSessionParams
