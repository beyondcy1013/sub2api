import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import { automaticDetectionMessages } from '@/modules/qualitycontrol/automaticMessages'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'

export default {
  adminAccessKeys,
  mcp,
  automaticDetection: automaticDetectionMessages.en,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
}
