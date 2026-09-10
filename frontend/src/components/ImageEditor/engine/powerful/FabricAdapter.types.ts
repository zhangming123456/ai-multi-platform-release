import type { FabricObject } from 'fabric'
import type { BaseLayer } from '../../ImageEditor.types'

export type TaggedObject = FabricObject & { dataLayerType?: BaseLayer['type']; dataSrc?: string }
