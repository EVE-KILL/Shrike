<script setup lang="ts">
interface TreeNode {
    id: number
    name: string
    slug: string
    parent_id: number | null
    has_types: boolean
    icon_id: number | null
    children: TreeNode[]
}

const props = defineProps<{
    nodes: TreeNode[]
    activePath: string[]
    depth?: number
    parentSlugPath?: string
}>()

const currentDepth = props.depth ?? 0
const parentPath = props.parentSlugPath ?? ''

const expanded = ref<Set<number>>(new Set())

// Auto-expand nodes that are in the active path
watchEffect(() => {
    if (props.activePath.length > currentDepth) {
        const targetSlug = props.activePath[currentDepth]
        const match = props.nodes.find(n => n.slug === targetSlug)
        if (match && match.children.length > 0) {
            expanded.value.add(match.id)
        }
    }
})

const toggle = (node: TreeNode) => {
    if (expanded.value.has(node.id)) {
        expanded.value.delete(node.id)
    } else {
        expanded.value.add(node.id)
    }
}

const slugPath = (node: TreeNode) => {
    return parentPath ? `${parentPath}/${node.slug}` : node.slug
}

const isActive = (node: TreeNode) => {
    const nodePath = slugPath(node)
    return props.activePath.join('/') === nodePath.split('/').slice(0, props.activePath.length).join('/')
        && props.activePath.length === nodePath.split('/').length
}
</script>

<template>
    <div :class="currentDepth > 0 ? 'ml-3 border-l border-white/[0.06] pl-2' : ''">
        <div v-for="node in nodes" :key="node.id">
            <div class="flex items-center gap-1 rounded-md transition-colors"
                :class="isActive(node) ? 'bg-blue-500/10' : 'hover:bg-blue-500/[0.04]'">
                <!-- Expand/collapse toggle -->
                <button v-if="node.children.length > 0"
                    class="flex-shrink-0 w-5 h-5 flex items-center justify-center text-gray-600 hover:text-gray-400"
                    @click.stop="toggle(node)">
                    <Icon :name="expanded.has(node.id) ? 'lucide:chevron-down' : 'lucide:chevron-right'" class="text-fine" />
                </button>
                <div v-else class="w-5 flex-shrink-0"></div>

                <!-- Group link -->
                <NuxtLink :to="`/market/${slugPath(node)}`"
                    class="flex-1 py-1 pr-2 text-xs truncate transition-colors"
                    :class="isActive(node) ? 'text-blue-400 font-medium' : 'text-gray-400 hover:text-blue-400'">
                    {{ node.name }}
                </NuxtLink>
            </div>

            <!-- Children (recursive) -->
            <MarketTreeSidebar
                v-if="node.children.length > 0 && expanded.has(node.id)"
                :nodes="node.children"
                :active-path="activePath"
                :depth="currentDepth + 1"
                :parent-slug-path="slugPath(node)"
            />
        </div>
    </div>
</template>
