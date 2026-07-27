<script setup lang="ts">
import type { WidgetConfig, DomainEntity } from '~/composables/useDomainConfig'

const props = defineProps<{
    widget: WidgetConfig
    statsEndpoint: string
    killlistEndpoint: string
    entities: readonly DomainEntity[]
}>()

const topBoxMap: Record<string, { title: string; dataType: string }> = {
    topCharacters: { title: 'Top Characters', dataType: 'characters' },
    topCorporations: { title: 'Top Corporations', dataType: 'corporations' },
    topAlliances: { title: 'Top Alliances', dataType: 'alliances' },
    topShips: { title: 'Top Ships', dataType: 'ships' },
    topSystems: { title: 'Top Systems', dataType: 'systems' },
    topRegions: { title: 'Top Regions', dataType: 'regions' },
}

const topBox = computed(() => topBoxMap[props.widget.type])
</script>

<template>
    <MostValuable v-if="widget.type === 'mostValuable'" :api-endpoint="statsEndpoint" />
    <KillList
        v-else-if="widget.type === 'killList'"
        :killlist-type="widget.killlistType || 'latest'"
        :api-endpoint="killlistEndpoint"
        :stream-topics="['all']"
    />
    <TopBox
        v-else-if="topBox"
        :title="topBox.title"
        :data-type="topBox.dataType"
        :limit="10"
        :days="7"
        :api-endpoint="statsEndpoint"
    />
    <EntityInfoWidget v-else-if="widget.type === 'entityInfo'" :entities="entities" />
    <TextBlockWidget v-else-if="widget.type === 'textBlock'" :content="widget.content || ''" />
    <DomainCampaignsWidget v-else-if="widget.type === 'campaigns'" />
</template>
