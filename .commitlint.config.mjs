export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'build',
        'chore',
        'ci',
        'docs',
        'feat',
        'fix',
        'perf',
        'refactor',
        'revert',
        'style',
        'test',
        'internal'
      ]
    ],
    'header-max-length': [0, 'always'],
    'body-max-line-length': [0, 'always']
  }
};
