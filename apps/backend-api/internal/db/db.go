package db

import (
	"ai-zombie-defense/backend-api/internal/db/generated"
)

type DBTX = generated.DBTX
type Queries = generated.Queries

func New() *Queries {
	return generated.New()
}

// Aliases for all generated types from the generated package
type GetPrestigeCosmeticsParams = generated.GetPrestigeCosmeticsParams
type GrantCosmeticToPlayerParams = generated.GrantCosmeticToPlayerParams
type CreateCurrencyTransactionParams = generated.CreateCurrencyTransactionParams
type GetCurrencyTransactionsByPlayerParams = generated.GetCurrencyTransactionsByPlayerParams
type GetCurrencyTransactionsByPlayerAndTypeParams = generated.GetCurrencyTransactionsByPlayerAndTypeParams
type AcceptFriendRequestParams = generated.AcceptFriendRequestParams
type CreateFriendRequestParams = generated.CreateFriendRequestParams
type DeclineFriendRequestParams = generated.DeclineFriendRequestParams
type GetFriendRequestParams = generated.GetFriendRequestParams
type ListFriendsRow = generated.ListFriendsRow
type ListPendingIncomingRow = generated.ListPendingIncomingRow
type ListPendingOutgoingRow = generated.ListPendingOutgoingRow
type CreateJoinTokenParams = generated.CreateJoinTokenParams
type GetAllTimeLeaderboardRow = generated.GetAllTimeLeaderboardRow
type GetDailyLeaderboardRow = generated.GetDailyLeaderboardRow
type GetWeeklyLeaderboardRow = generated.GetWeeklyLeaderboardRow
type CreateLoadoutParams = generated.CreateLoadoutParams
type DeleteLoadoutCosmeticBySlotParams = generated.DeleteLoadoutCosmeticBySlotParams
type GetLoadoutCosmeticBySlotParams = generated.GetLoadoutCosmeticBySlotParams
type GetLoadoutCosmeticsRow = generated.GetLoadoutCosmeticsRow
type InsertLoadoutCosmeticParams = generated.InsertLoadoutCosmeticParams
type UpdateLoadoutActiveParams = generated.UpdateLoadoutActiveParams
type CreateLootTableEntryParams = generated.CreateLootTableEntryParams
type GetLootTableEntriesWithCosmeticDetailsRow = generated.GetLootTableEntriesWithCosmeticDetailsRow
type UpdateLootTableEntryParams = generated.UpdateLootTableEntryParams
type CreateLootTableParams = generated.CreateLootTableParams
type UpdateLootTableParams = generated.UpdateLootTableParams
type CreateMatchParams = generated.CreateMatchParams
type GetPlayerMatchHistoryParams = generated.GetPlayerMatchHistoryParams
type GetPlayerMatchHistoryRow = generated.GetPlayerMatchHistoryRow
type UpdateMatchOutcomeParams = generated.UpdateMatchOutcomeParams
type CosmeticItem = generated.CosmeticItem
type CurrencyTransaction = generated.CurrencyTransaction
type Friend = generated.Friend
type JoinToken = generated.JoinToken
type LeaderboardEntry = generated.LeaderboardEntry
type Loadout = generated.Loadout
type LoadoutCosmetic = generated.LoadoutCosmetic
type LootTable = generated.LootTable
type LootTableEntry = generated.LootTableEntry
type Match = generated.Match
type Player = generated.Player
type PlayerCosmetic = generated.PlayerCosmetic
type PlayerMatchStat = generated.PlayerMatchStat
type PlayerProgression = generated.PlayerProgression
type PlayerSetting = generated.PlayerSetting
type Server = generated.Server
type ServerFavorite = generated.ServerFavorite
type Session = generated.Session
type CreatePlayerParams = generated.CreatePlayerParams
type UpdatePlayerLastLoginParams = generated.UpdatePlayerLastLoginParams
type UpdatePlayerPasswordParams = generated.UpdatePlayerPasswordParams
type UpdatePlayerProfileParams = generated.UpdatePlayerProfileParams
type GetPlayerCosmeticParams = generated.GetPlayerCosmeticParams
type GetPlayerCosmeticRow = generated.GetPlayerCosmeticRow
type GetPlayerCosmeticsRow = generated.GetPlayerCosmeticsRow
type CreatePlayerMatchStatsParams = generated.CreatePlayerMatchStatsParams
type GetPlayerMatchStatsParams = generated.GetPlayerMatchStatsParams
type AddDataCurrencyParams = generated.AddDataCurrencyParams
type IncrementExperienceParams = generated.IncrementExperienceParams
type IncrementMatchStatsParams = generated.IncrementMatchStatsParams
type SetDataCurrencyParams = generated.SetDataCurrencyParams
type UpdateLevelParams = generated.UpdateLevelParams
type UpdatePlayerProgressionParams = generated.UpdatePlayerProgressionParams
type UpsertPlayerSettingsParams = generated.UpsertPlayerSettingsParams
type AddFavoriteParams = generated.AddFavoriteParams
type GetFavoriteParams = generated.GetFavoriteParams
type ListPlayerFavoritesRow = generated.ListPlayerFavoritesRow
type RemoveFavoriteParams = generated.RemoveFavoriteParams
type CreateServerParams = generated.CreateServerParams
type ListActiveServersParams = generated.ListActiveServersParams
type UpdateServerHeartbeatParams = generated.UpdateServerHeartbeatParams
type CreateSessionParams = generated.CreateSessionParams
