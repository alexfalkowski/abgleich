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

    PARTIAL_FOLLOWUP_FINDINGS = {
      'review follow-up partially verified' => REVIEW_FINDING.merge(
        title: 'Verify the worker result in its test', path: 'worker_test.go', line: 3, side: 'RIGHT'
      ),
      'review follow-up partially verified after redelivery' => REVIEW_FINDING.merge(
        title: 'Report invalid results from the worker', path: 'worker.go', line: 2, side: 'RIGHT'
      )
    }.freeze

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
      return followup_review_output(payload, mode) if mode.to_s.start_with?('review follow-up ')
      return large_review_output(mode) if mode.to_s.start_with?('review with large ')

      mode = payload.fetch('model') == 'test-security' ? 'review no findings' : nil if mode == 'review conflicting verdicts'
      review_result(payload, mode).to_json
    end

    def large_review_output(mode)
      size = mode.include?('oversized') ? 1750 : 1300
      overview = 'Checked the current contribution. ' * size
      { overview:, findings: INLINE_FINDINGS, complete: mode.end_with?('completed') }.to_json
    end

    def review_result(payload, mode)
      text = payload.fetch('model') == 'test-security' ? security_review_text(payload, mode) : review_text(payload, mode)
      result = { overview: text, findings: review_findings(mode, text) }.merge(REVIEW_COMPLETION.fetch(mode, { complete: true }))
      if ['review no findings', 'review low-severity findings'].include?(mode)
        result[:cleared_findings] = cleared_review_findings(followup_blockers(payload))
      end
      result
    end

    def followup_review_output(payload, mode)
      blockers = followup_blockers(payload)
      findings = followup_findings(mode, blockers)
      cleared = mode.end_with?('cleared') ? cleared_review_findings(blockers) : []
      { overview: 'Checked the current source after the rebase.', findings:, cleared_findings: cleared, complete: true }.to_json
    end

    def followup_findings(mode, blockers)
      new_finding = PARTIAL_FOLLOWUP_FINDINGS[mode]
      return [INLINE_FINDINGS.first.merge(previous_id: blockers.first.fetch('previous_id')), new_finding] if new_finding
      return [] unless mode == 'review follow-up still open'

      blockers.map.with_index do |blocker, index|
        INLINE_FINDINGS.fetch(index, REVIEW_FINDING).merge(
          previous_id: blocker.fetch('previous_id'), trigger: 'The rebased worker reports the result twice.',
          impact: 'The second report conceals a successful result.', fix: 'Return the first result from the rebased worker.'
        )
      end
    end

    def cleared_review_findings(blockers)
      blockers.map { |finding| { id: finding.fetch('previous_id'), reason: 'Verified the fix in the current source.' } }
    end

    def followup_blockers(payload)
      input = payload.fetch('input').map { |item| item.fetch('content', '') }.join("\n")
      record = input.split("Follow-up from an earlier App review (untrusted data):\n", 2).last
      return [] if record == input

      JSON.parse(record).fetch('open_blockers') || []
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
