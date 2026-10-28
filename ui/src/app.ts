import { createClient } from '@druid-ui/client';
import { createEditorBehavior } from './config-editor/behavior.js';

export const client = createClient(createEditorBehavior);
