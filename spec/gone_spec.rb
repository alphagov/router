require 'spec_helper'

describe "Gone endpoints" do

  before :each do
    add_gone_route("/foo")
    add_gone_route("/bar", :prefix => true)
    reload_routes
  end

  it "should support an exact gone route" do
    response = router_request("/foo")
    expect(response.code).to eq(410)

    response = router_request("/foo/bar")
    expect(response.code).to eq(404)
  end

  it "should support a prefix gone route" do
    response = router_request("/bar")
    expect(response.code).to eq(410)

    response = router_request("/bar/baz")
    expect(response.code).to eq(410)
  end
end
