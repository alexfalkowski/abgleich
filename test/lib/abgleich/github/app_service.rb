# frozen_string_literal: true

module Abgleich
  # App and comment endpoints extend the same local fake GitHub service.
  class GitHubService
    get '/app/installations/:installation' do
      halt 404 unless params.fetch('installation').to_i == INSTALLATION_ID
      { id: INSTALLATION_ID, app_id: APP_ID, account: { login: Abgleich.github.authenticated_owner } }.to_json
    end

    post '/app/installations/:installation/access_tokens' do
      halt 404 unless params.fetch('installation').to_i == INSTALLATION_ID
      GitHubService.record_installation_request
      halt 503 if Abgleich.github.failure == 'installation token HTTP'
      sleep 3 if Abgleich.github.failure == 'installation token timeout'
      Abgleich.github.create_installation_token.to_json
    end

    get '/installation/repositories' do
      page = params.fetch('page', '1').to_i
      halt 503 if Abgleich.github.failure == 'discovery' || (page > 1 && Abgleich.github.failure == 'discovery page')
      halt 200, '{}' if Abgleich.github.failure == 'incomplete discovery'

      repositories, more = Abgleich.github.repositories(page)
      headers 'Link' => "<http://localhost:11001/installation/repositories?page=#{page + 1}>; rel=\"next\"" if more
      { repositories: }.to_json
    end

    get '/repos/:owner/:repository/issues/comments/:id' do
      halt 503 if Abgleich.github.failure == 'comment HTTP'
      sleep 3 if Abgleich.github.failure == 'comment timeout'
      Abgleich.github.comment(params.fetch('id').to_i).to_json
    end

    get '/repos/:owner/:repository/collaborators/:user/permission' do
      halt 503 if Abgleich.github.failure == 'permission HTTP'
      Abgleich.github.comment_permission.to_json
    end

    post '/repos/:owner/:repository/issues/comments/:id/reactions' do
      failure = Abgleich.github.failure
      content = JSON.parse(request.body.read).fetch('content')
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
