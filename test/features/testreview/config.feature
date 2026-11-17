@manual @commands @testreview
Feature: Choose whether maintainers can request test reviews
  Test reviews are independently enabled with their own model and schedule.

  Background:
    Given AI test reviews are configured for this repository

  Scenario: Refuse a test review without a model
    Given the test review configuration has "no model"
    When the operator starts the worker
    Then the worker should refuse to start

  Scenario: Leave disabled test review commands unused
    Given the test review configuration has "disabled reviews"
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab testreview"
    Then the command comment should show "receipt and rejection" feedback
    And the AI service should remain unused

  Scenario: Request a test review with the default schedule
    Given the test review configuration has "the default schedule"
    And a pull request from "the repository" for branch "contribution" is "waiting for checks"
    When a maintainer with "write" access comments "/ab testreview"
    Then the test review should identify the request and revisions
