# frozen_string_literal: true

module Abgleich
  # App and comment endpoints extend the same local fake GitHub service.
  class GitHubService
    get '/app/installations/:installation' do
      halt 404 unless params.fetch('installation').to_i == INSTALLATION_ID
      GitHubService.record_installation_check
      installation = { id: INSTALLATION_ID, app_id: APP_ID, account: { login: Abgleich.github.authenticated_owner } }
      case Abgleich.github.failure
      when 'empty installation' then halt 200, ''
      when 'null installation' then halt 200, 'null'
      when 'mismatched installation' then installation[:id] += 1
      when 'mismatched app' then installation[:app_id] += 1
      when 'suspended installation' then installation[:suspended_at] = '2026-01-01T00:00:00Z'
      end
      installation.to_json
    end

    post '/app/installations/:installation/access_tokens' do
      halt 404 unless params.fetch('installation').to_i == INSTALLATION_ID
      GitHubService.record_installation_request
      if Abgleich.github.progress.fetch(:scans).positive?
        case Abgleich.github.failure
        when 'installation token renewal rate limit'
          headers 'Retry-After' => '60'
          halt 429, { message: 'Rate limit exceeded' }.to_json
        when 'installation token renewal HTTP'
          halt 503
        end
      end
      halt 503 if Abgleich.github.failure == 'installation token HTTP'
      sleep 3 if Abgleich.github.failure == 'installation token timeout'
      Abgleich.github.create_installation_token.to_json
    end

    get '/installation/repositories' do
      page = params.fetch('page', '1').to_i
      GitHubService.record_repository_discovery if page == 1
      halt 503 if Abgleich.github.failure == 'discovery' || (page > 1 && Abgleich.github.failure == 'discovery page')
      halt 200, '{}' if Abgleich.github.failure == 'incomplete discovery'

      repositories, more = Abgleich.github.repositories(page)
      headers 'Link' => "<http://localhost:11001/github/installation/repositories?page=#{page + 1}>; rel=\"next\"" if more
      { repositories: }.to_json
    end

    get '/repos/:owner/:repository/issues/comments/:id' do
      halt 404 if Abgleich.github.failure == 'review deleted comment'
      halt 503 if Abgleich.github.failure == 'comment HTTP'
      sleep 3 if Abgleich.github.failure == 'comment timeout'
      comment = Abgleich.github.comment(params.fetch('id').to_i)
      comment['updated_at'] = '2026-01-01T00:00:06Z' if Abgleich.github.failure == 'final authorization:an edited comment'
      comment.to_json
    end

    get '/repos/:owner/:repository/collaborators/:user/permission' do
      halt 503 if Abgleich.github.failure == 'permission HTTP'
      permission = Abgleich.github.comment_permission
      permission[:permission] = 'read' if Abgleich.github.failure == 'final authorization:revoked write access'
      permission.to_json
    end

    post '/repos/:owner/:repository/issues/comments/:id/reactions' do
      failure = Abgleich.github.failure
      content = JSON.parse(request.body.read).fetch('content')
      sleep 1.2 if failure == 'slow final reaction' && content != 'eyes'
      if failure == 'reaction timeout'
        sleep 3
        halt 503
      end
      halt 503 if failure == 'reaction HTTP'
      halt 201, { id: 123, content: 'heart' }.to_json if failure == 'mismatched reaction'
      halt 201, { content: }.to_json if failure == 'incomplete reaction'

      reaction, created = Abgleich.github.react_to_comment(params.fetch('id').to_i, content)
      status(created ? 201 : 200)
      reaction.to_json
    end

    post '/repos/:owner/:repository/issues/:number/comments' do
      Abgleich.github.record_reply_request
      failure = Abgleich.github.failure
      halt 503 if failure == 'reply HTTP'
      halt 201, '{' if failure == 'malformed reply'
      halt 201, '{}' if failure == 'incomplete reply'
      sleep 3 if failure == 'reply timeout'
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      reply = Abgleich.github.publish_reply(repository, params.fetch('number').to_i, JSON.parse(request.body.read).fetch('body'))
      halt 503 if failure == 'lost reply response'
      reply[:issue_url] = 'http://localhost:11001/github/repos/someone/other/issues/999' if failure == 'mismatched reply'
      reply[:body] = 'different body' if failure == 'wrong reply body'
      reply.delete(:id) if failure == 'missing reply ID'
      status 201
      reply.to_json
    end

    get '/repos/:owner/:repository/issues/:number/comments' do
      page = params.fetch('page', '1').to_i
      per_page = params.fetch('per_page', '30').to_i
      halt 503 if Abgleich.github.failure == "summary comments page:#{page}"

      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      number = params.fetch('number')
      comments = Abgleich.github.issue_comments(repository, number.to_i)
      if page * per_page < comments.length
        url = "http://localhost:11001/github/repos/#{repository}/issues/#{number}/comments"
        headers 'Link' => "<#{url}?page=#{page + 1}&per_page=#{per_page}>; rel=\"next\""
      end
      comments = comments.slice((page - 1) * per_page, per_page).to_a.map(&:dup)
      if Abgleich.github.failure == 'summary comment wrong PR'
        comments.last[:issue_url] = 'http://localhost:11001/github/repos/someone/other/issues/999'
      end
      comments.to_json
    end

    get '/repos/:owner/:repository/pulls/:number/files' do
      page = params.fetch('page', '1').to_i
      halt 503 if Abgleich.github.failure == "summary files page:#{page}"

      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      files = Abgleich.github.summary_files
      files.each { |file| file[:patch] = GitHubRepository::Reviews::PATCH } if Abgleich.github.file_mode == 'code review'
      Abgleich.github.apply_collection_change
      number = params.fetch('number')
      if page * 2 < files.length
        headers 'Link' => "<http://localhost:11001/github/repos/#{repository}/pulls/#{number}/files?page=#{page + 1}>; rel=\"next\""
      end
      files.slice((page - 1) * 2, 2).to_a.to_json
    end

    post '/repos/:owner/:repository/issues/:number/labels' do
      halt 503 if Abgleich.github.failure == 'label HTTP'
      sleep 1 if Abgleich.github.failure == 'slow label'
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      labels = JSON.parse(request.body.read)
      result = Abgleich.github.request_labels(repository, params.fetch('number').to_i, labels)
      halt 503 if Abgleich.github.failure == 'lost label response'
      result.to_json
    end

    private

    def change_status_state(repository, rechecked, failure)
      if failure&.start_with?('status settings:')
        setting = failure.delete_prefix('status settings:').to_sym
        Abgleich.github.add_repository(repository, **{ setting => false })
        Abgleich.github.fail_with(nil)
      elsif rechecked && failure&.start_with?('status authorization:')
        Abgleich.github.fail_with(failure.sub('status authorization:', 'final authorization:'))
      end
    end

    def authenticate_request
      return authenticate_app if request.path_info.start_with?('/app/')

      expected = "token #{Abgleich.github.installation_token}"
      halt 401, { message: 'Bad credentials' }.to_json unless request.env['HTTP_AUTHORIZATION'] == expected
    end

    def authenticate_app
      jwt = request.env['HTTP_AUTHORIZATION'].to_s.delete_prefix('Bearer ')
      claims, = JWT.decode(jwt, Abgleich.app_public_key, true, algorithm: 'RS256')
      halt 401 unless claims.fetch('iss') == APP_ID.to_s
    rescue JWT::DecodeError
      halt 401
    end
  end
end
