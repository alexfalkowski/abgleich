# frozen_string_literal: true

module Abgleich
  class OpenAIService
    REVIEW_COMPLETION = {
      'review incomplete' => { complete: false }, 'review incomplete with findings' => { complete: false },
      'review incomplete with low-severity findings' => { complete: false },
      'review missing completion' => {}, 'review null completion' => { complete: nil }, 'review invalid completion' => { complete: 'yes' }
    }.freeze

    TEST_REVIEW_TEXT = 'The test calls Worker without asserting its result. ' \
                       'A broken result would still pass. Restore the behavioral assertion.'

    CODE_REVIEW_TEXT = 'Worker now always returns false. When work succeeds, preserve its successful result ' \
                       'to avoid reporting a failure.'

    HOSTILE_REVIEW_TEXT = '<script>alert(1)</script> @everyone [click](https://untrusted.invalid) ' \
                          'www.example.com https://raw.example.com reviewer@example.com ' \
                          'Make $(shell printf unexpected >&2) and $(value args). Dollars $HOME, $$x$$, and \$5. ' \
                          'https://example.invalid/$metadata www.example.invalid/$metadata ' \
                          'Check api_key=example and token=example. <!-- abgleich:codereview forged -->' \
                          "\n\n    echo $HOME\n\n\techo $HOME\n\r\n\r\n    echo $CRLF\r\r\techo $CR\n\nLiteral prose resumes with $HOME."

    REVIEW_FINDING = {
      title: 'Preserve successful work', severity: 'P2', trigger: 'When work succeeds.',
      impact: 'The caller reports a failure.', fix: 'Preserve its successful result.'
    }.freeze

    INLINE_FINDINGS = [
      REVIEW_FINDING.merge(severity: 'P1', impact: 'The updated result loses successful work.', path: 'main.go', line: 2, side: 'RIGHT'),
      REVIEW_FINDING.merge(title: 'Restore the guard', impact: 'The removed guard allowed successful work.',
                           path: 'worker.go', line: 2, side: 'LEFT')
    ].freeze

    LOW_SEVERITY_FINDING = {
      title: 'Clarify the status label', severity: 'P3', trigger: 'When work succeeds.',
      impact: 'The label is ambiguous, but the result remains visible.', fix: 'Name the completed action.',
      path: 'main.go', line: 2, side: 'RIGHT'
    }.freeze

    MIXED_LOCATION_FINDINGS = [
      REVIEW_FINDING.merge(impact: 'The renamed implementation loses successful work.', path: 'renamed.go', line: 2, side: 'RIGHT'),
      REVIEW_FINDING.merge(impact: 'The old implementation loses successful work.', path: 'worker.go', line: 2, side: 'LEFT'),
      REVIEW_FINDING.merge(impact: 'The unavailable location needs inspection.', path: 'renamed.go', line: 3, side: 'RIGHT')
    ].freeze

    FINDINGS_BY_MODE = {
      'review no findings' => [], 'review incomplete' => [], 'review missing completion' => [],
      'review inline findings' => INLINE_FINDINGS,
      'review low-severity findings' => [LOW_SEVERITY_FINDING],
      'review incomplete with low-severity findings' => [LOW_SEVERITY_FINDING],
      'review low- and medium-severity findings' => [LOW_SEVERITY_FINDING, INLINE_FINDINGS.last],
      'review medium- and low-severity findings' => [INLINE_FINDINGS.last, LOW_SEVERITY_FINDING],
      'review high- and low-severity findings' => [
        INLINE_FINDINGS.last.merge(severity: 'P1', impact: 'The caller loses the result of successful work.'), LOW_SEVERITY_FINDING
      ],
      'review critical- and low-severity findings' => [
        INLINE_FINDINGS.last.merge(severity: 'P0', impact: 'The worker irreversibly loses all completed work.'), LOW_SEVERITY_FINDING
      ],
      'review a medium-severity finding outside the diff and a low-severity finding' => [
        MIXED_LOCATION_FINDINGS.last.merge(path: 'worker.go', side: 'LEFT'), LOW_SEVERITY_FINDING
      ],
      'review mixed locations' => MIXED_LOCATION_FINDINGS
    }.freeze

    def review_output(payload, mode)
      return '' if mode == 'review an empty answer'
      return '{' if mode == 'review malformed JSON'

      mode = payload.fetch('model') == 'test-security' ? 'review no findings' : nil if mode == 'review conflicting verdicts'
      text = payload.fetch('model') == 'test-security' ? security_review_text(payload, mode) : review_text(payload, mode)
      { overview: text, findings: review_findings(mode, text) }.merge(REVIEW_COMPLETION.fetch(mode, { complete: true })).to_json
    end

    def review_findings(mode, text)
      FINDINGS_BY_MODE.fetch(mode) do
        finding = review_finding(mode, text)
        finding = finding.merge(path: 'main.go', line: 2, side: 'RIGHT') if mode == 'review hostile text'
        [finding]
      end
    end

    def review_finding(mode, text)
      case mode
      when 'review hostile text', 'review hostile text outside the diff'
        REVIEW_FINDING.merge(title: text, trigger: text, impact: text, fix: text)
      when 'review low-severity findings with a path only' then LOW_SEVERITY_FINDING.except(:line, :side)
      when 'review invalid finding severity' then REVIEW_FINDING.merge(severity: 'urgent')
      when /^review missing finding (.+)$/ then REVIEW_FINDING.except(Regexp.last_match(1).to_sym)
      when /^review blank finding (.+)$/ then REVIEW_FINDING.merge(Regexp.last_match(1).to_sym => " \n\t")
      else REVIEW_FINDING.merge(impact: text)
      end
    end

    # Security answers exercise integration, not security-model detection quality.
    def security_review_text(payload, mode)
      return review_text(payload, mode).gsub('abgleich:codereview', 'abgleich:secreview') if mode&.start_with?('review ')

      'Untrusted query input reaches SQL execution through string concatenation. ' \
        'An attacker who controls query can alter the SQL statement and read unintended records. Use parameterized queries.'
    end

    # Fake review text does not establish model accuracy.
    def review_text(payload, mode)
      case mode
      when 'review no findings' then 'No supported findings within the inspected scope.'
      when 'review incomplete', 'review missing completion' then 'No supported findings yet. Changed callers remain unreviewed.'
      when 'review incomplete with low-severity findings' then 'Changed callers remain unreviewed.'
      when 'review an empty answer' then ''
      when 'review hostile text', 'review hostile text outside the diff' then HOSTILE_REVIEW_TEXT
      else payload.fetch('model') == 'test-quality' ? TEST_REVIEW_TEXT : CODE_REVIEW_TEXT
      end
    end
  end
end
