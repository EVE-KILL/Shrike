// State, validation and wire conversion ported from campaigncreator.vue.
import {
  CAMPAIGN_MAX_LOCATION_SYSTEMS,
  CAMPAIGN_MAX_LOCATION_CONSTELLATIONS,
  CAMPAIGN_MAX_LOCATION_REGIONS,
  CAMPAIGN_MAX_PUBLIC_WINDOW_DAYS,
  CAMPAIGN_MAX_PRIVATE_WINDOW_DAYS,
  CAMPAIGN_PRIZE_DEFAULT_SPLITS,
} from "./campaign-data.js";
import type {CampaignEntity,CampaignLocation} from "./shared/campaign-model";
export type CampaignMode="conflict"|"single"|"area";
export type CampaignLocationType="system"|"constellation"|"region";
export interface CampaignDraft {name:string;description:string;mode:CampaignMode;visibility:number;startTime:string;endTime:string;ongoing:boolean;sides:Array<{name:string;entities:CampaignEntity[]}>;locations:CampaignLocation[];allowed:CampaignEntity[];prizeEnabled:boolean;prizeMetric:number;prizeWinnerCount:number;prizePercentages:Array<number|string>;prizeInitialContribution:string;fundingRequestId:string}
export interface CampaignWallet {availableBalance?:string;balance?:string}

export const CAMPAIGN_LOCATION_CAPS = {
  system: CAMPAIGN_MAX_LOCATION_SYSTEMS,
  constellation: CAMPAIGN_MAX_LOCATION_CONSTELLATIONS,
  region: CAMPAIGN_MAX_LOCATION_REGIONS,
};
export const campaignVisibleSides = (state:CampaignDraft) =>
  state.mode === "single" ? state.sides.slice(0, 1) : state.sides;
export const campaignPicked = (state:CampaignDraft,type:string,id:number) =>
  campaignVisibleSides(state).some((side) =>
    side.entities.some((entity) => entity.type === type && entity.id === id),
  );
export const campaignLocationIds = (state:CampaignDraft,type:string) =>
  state.locations
    .filter((location) => location.type === type)
    .map((location) => location.id);
export const campaignTimeISO = (value:string) => new Date(value + "Z").toISOString();
export function newCampaignDraft(start:string):CampaignDraft {
  return {
    name: "",
    description: "",
    mode: "conflict",
    visibility: 0,
    startTime: start,
    endTime: "",
    ongoing: true,
    sides: [
      { name: "Side 1", entities: [] },
      { name: "Side 2", entities: [] },
    ],
    locations: [],
    allowed: [],
    prizeEnabled: false,
    prizeMetric: 0,
    prizeWinnerCount: 3,
    prizePercentages: [...CAMPAIGN_PRIZE_DEFAULT_SPLITS[3]],
    prizeInitialContribution: "",
    fundingRequestId: "",
  };
}
export const campaignHasName = (state:CampaignDraft) => state.name.trim().length >= 3;
export const campaignHasParticipants = (state:CampaignDraft) =>
  state.mode === "area"
    ? state.locations.length > 0
    : campaignVisibleSides(state).length >= 1 &&
      campaignVisibleSides(state).every((side) => side.entities.length >= 1);
export function campaignTimeError(state:CampaignDraft, now = Date.now()) {
  if (!state.startTime) return "Start time is required";
  if (!state.ongoing && !state.endTime)
    return "Set an end time or mark the campaign as ongoing";
  const start = new Date(state.startTime + "Z").getTime(),
    end = state.ongoing ? now : new Date(state.endTime + "Z").getTime();
  if (!state.ongoing && state.endTime && end <= start)
    return "End must be after start";
  const maxDays =
    state.visibility === 0
      ? CAMPAIGN_MAX_PUBLIC_WINDOW_DAYS
      : CAMPAIGN_MAX_PRIVATE_WINDOW_DAYS;
  if ((end - start) / 86400000 > maxDays)
    return `${state.visibility === 0 ? "Public" : "Non-public"} campaigns may cover at most ${maxDays} days`;
  return null;
}
export function campaignPrizeError(
  state:CampaignDraft,
  wallet:CampaignWallet|null = null,
  walletPending = false,
  now = Date.now(),
) {
  if (!state.prizeEnabled) return null;
  if (state.ongoing || !state.endTime)
    return "Prize campaigns require an end time";
  if (new Date(state.endTime + "Z").getTime() <= now)
    return "Prize campaigns must end in the future";
  if (
    state.prizePercentages.some(
      (value) => !Number.isInteger(Number(value)) || Number(value) < 1,
    )
  )
    return "Every winning place must receive at least 1%";
  if (
    state.prizePercentages.reduce<number>(
      (sum, value) => sum + (Number(value) || 0),
      0,
    ) !== 100
  )
    return "Payout percentages must total exactly 100%";
  const raw = state.prizeInitialContribution.trim(),
    amount = Number(raw || 0);
  if (raw && !/^\d+(?:\.\d{1,2})?$/.test(raw))
    return "Initial funding must be a positive ISK amount with at most two decimals";
  if (amount > 0 && walletPending)
    return "Your EK Wallet balance is still loading";
  const available = campaignAvailableBalance(wallet);
  if (amount > 0 && (available === null || amount > available))
    return "Initial funding exceeds your available EK Wallet balance";
  return null;
}
export function campaignAvailableBalance(wallet:CampaignWallet|null) {
  const value = Number(wallet?.availableBalance ?? wallet?.balance);
  return Number.isFinite(value) ? value : null;
}
export function campaignChecklist(state:CampaignDraft,wallet:CampaignWallet|null,pending:boolean, now = Date.now()) {
  const hasName = campaignHasName(state),
    hasParticipants = campaignHasParticipants(state),
    timeError = campaignTimeError(state, now),
    prizeError = campaignPrizeError(state,wallet,pending,now);
  const rows = [
    {
      label: "Name the campaign",
      ok: hasName,
      detail: hasName ? null : "At least 3 characters",
    },
  ];
  if (state.mode === "area")
    rows.push({
      label: "Pick a battlefield",
      ok: state.locations.length > 0,
      detail: state.locations.length
        ? null
        : "At least one region, constellation or system",
    });
  else
    rows.push({
      label:
        campaignVisibleSides(state).length > 1
          ? "Fill every side"
          : "Pick who to track",
      ok: hasParticipants,
      detail: hasParticipants ? null : "Each side needs at least one entity",
    });
  rows.push({ label: "Set the time range", ok: !timeError, detail: timeError });
  if (state.prizeEnabled)
    rows.push({
      label: "Configure the prize pool",
      ok: !prizeError,
      detail: prizeError,
    });
  return rows;
}
export const campaignCanSubmit = (state:CampaignDraft,wallet:CampaignWallet|null,pending:boolean,now = Date.now()) =>
  campaignHasName(state) &&
  campaignHasParticipants(state) &&
  !campaignTimeError(state, now) &&
  !campaignPrizeError(state,wallet,pending,now);
export function campaignCreateDocument(state:CampaignDraft) {
  const funding = state.prizeInitialContribution.trim();
  return {
    name: state.name.trim(),
    description: state.description.trim() || null,
    startTime: campaignTimeISO(state.startTime),
    endTime:
      state.ongoing || !state.endTime ? null : campaignTimeISO(state.endTime),
    location: {
      systemIds: campaignLocationIds(state, "system"),
      constellationIds: campaignLocationIds(state, "constellation"),
      regionIds: campaignLocationIds(state, "region"),
    },
    sides:
      state.mode === "area"
        ? []
        : campaignVisibleSides(state).map((side) => ({
            name: side.name.trim(),
            entities: side.entities.map((entity) => ({
              type: entity.type,
              id: entity.id,
            })),
          })),
    visibility: state.visibility,
    allowedEntities:
      state.visibility === 2 && state.mode === "area"
        ? state.allowed.map((entity) => ({
            type: entity.type,
            id: entity.id,
            name: entity.name,
          }))
        : [],
    prizePool: state.prizeEnabled
      ? {
          enabled: true,
          metric: state.prizeMetric,
          winnerCount: state.prizeWinnerCount,
          payoutPercentages: state.prizePercentages.map(Number),
          initialContribution: funding || "0",
          fundingRequestId:
            Number(funding || 0) > 0 ? state.fundingRequestId : undefined,
        }
      : { enabled: false },
  };
}
