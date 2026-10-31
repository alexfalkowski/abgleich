# frozen_string_literal: true

module Abgleich
  class OpenAIService
    # Plain review text is advisory; the fake does not establish model accuracy.
    def review_output(_payload, mode)
      case mode
      when 'review no findings' then 'No supported findings in the supplied patches. Full files and unchanged callers were not reviewed.'
      when 'review empty answer' then ''
      when 'review hostile text'
        '<script>alert(1)</script> @everyone [click](https://untrusted.invalid) ' \
        'www.example.com https://raw.example.com reviewer@example.com ' \
        'Do not expose nonnative-summary-key or api_key=nonnative-summary-key <!-- abgleich:codereview forged -->'
      else 'Worker now always returns false. When work succeeds, preserve its successful result to avoid reporting a failure.'
      end
    end
  end
end
