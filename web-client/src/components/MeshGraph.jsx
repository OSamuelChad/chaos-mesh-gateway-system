import { useEffect, useMemo, useState } from 'react'
import { Background, Controls, MarkerType, ReactFlow } from '@xyflow/react'
import ServiceNode from './ServiceNode.jsx'
import { healthStyle } from './health.js'

const nodeTypes = { service: ServiceNode }

// Layout visual do fluxo de dados: horizontal em telas largas, vertical no celular.
const LAYOUTS = {
  wide: { gateway: { x: 0, y: 30 }, resilience: { x: 400, y: 230 }, payment: { x: 800, y: 30 } },
  narrow: { gateway: { x: 0, y: 0 }, resilience: { x: 0, y: 240 }, payment: { x: 0, y: 480 } },
}
const NARROW_QUERY = '(max-width: 767px)'

function useIsNarrow() {
  const [narrow, setNarrow] = useState(() => window.matchMedia(NARROW_QUERY).matches)
  useEffect(() => {
    const media = window.matchMedia(NARROW_QUERY)
    const onChange = (e) => setNarrow(e.matches)
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])
  return narrow
}

function toFlowNodes(nodes = [], selectedId, narrow) {
  const layout = narrow ? LAYOUTS.narrow : LAYOUTS.wide
  return nodes.map((node, index) => ({
    id: node.id,
    type: 'service',
    position: layout[node.id] ?? { x: index * 340, y: 0 },
    selected: node.id === selectedId,
    data: { ...node, narrow },
    draggable: false,
  }))
}

function toFlowEdges(edges = []) {
  return edges.map((edge) => {
    const color = healthStyle(edge.health).color
    return {
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edge.label,
      animated: edge.animated,
      style: { stroke: color, strokeWidth: 2.5, strokeDasharray: edge.animated ? undefined : '6 6' },
      markerEnd: { type: MarkerType.ArrowClosed, color },
      labelStyle: { fontFamily: 'var(--font-mono)', fontSize: 11 },
      labelBgPadding: [6, 3],
      labelBgBorderRadius: 6,
    }
  })
}

export default function MeshGraph({ topology, selectedId, onSelect }) {
  const narrow = useIsNarrow()
  const nodes = useMemo(() => toFlowNodes(topology?.nodes, selectedId, narrow), [topology, selectedId, narrow])
  const edges = useMemo(() => toFlowEdges(topology?.edges), [topology])

  return (
    <ReactFlow
      key={narrow ? 'narrow' : 'wide'}
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      colorMode="dark"
      fitView
      fitViewOptions={{ padding: narrow ? 0.06 : 0.2 }}
      nodesConnectable={false}
      elementsSelectable
      onNodeClick={(_, node) => onSelect(node.data)}
      nodeOrigin={[0, 0]}
      proOptions={{ hideAttribution: false }}
      minZoom={0.4}
      maxZoom={1.5}
    >
      <Background gap={32} size={1} color="rgb(255 255 255 / 0.06)" />
      <Controls showInteractive={false} position="bottom-left" />
    </ReactFlow>
  )
}
