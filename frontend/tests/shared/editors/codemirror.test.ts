import { describe, expect, it } from 'vitest';
import { basicSetup } from 'codemirror';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';

describe('CodeMirror 6 editor foundation', () => {
  it('mounts the editor and applies an in-memory edit without losing Unicode or base64 text', () => {
    const source = 'dn: cn=José,dc=example\ncn: José\njpegPhoto:: AP/+\n';
    const parent = document.createElement('div');
    document.body.append(parent);
    const state = EditorState.create({ doc: source, extensions: [basicSetup] });
    const editor = new EditorView({ state, parent });

    try {
      expect(parent.querySelector('.cm-editor')).not.toBeNull();
      expect(editor.state.doc.toString()).toBe(source);
      editor.dispatch({
        changes: { from: source.length, insert: 'description: 日本語\n' },
      });
      expect(editor.state.doc.toString()).toBe(`${source}description: 日本語\n`);
    } finally {
      editor.destroy();
      parent.remove();
    }
  });
});
