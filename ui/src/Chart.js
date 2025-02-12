import { FlowChartWithState } from "@mrblenny/react-flow-chart";
import React from 'react'
import { processData } from './Data'

import Container from 'react-bootstrap/Container'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'

const NodeInnerCustom = ({ node, children, ...otherProps }) => {
  var className = "node";

  if (node.properties.response !== 200 && node.properties.response !== 0) {
    className = "node-error";
  }

  const ips = [];

  // if the node has ip addresses create individual elements for them
  if(node.properties.ip_addresses !== undefined) {
    for(var n=0;n<node.properties.ip_addresses.length;n++){
      ips.push(<div key={n}>{node.properties.ip_addresses[n]}</div>);
    }
  }

  return (
    <Container {...otherProps} className={className} key={node.properties.name}>
      <Row>
        <Col className="node-header">{node.properties.name}</Col>
      </Row>
      <Row>
        <Col className="node-uri">{node.properties.upstream_address}</Col>
      </Row>
      <Row>
        <Col>
    <Container>
      { node.properties.hostname && node.properties.hostname.length > 0 && (
        <Row>
          <Col className="node-key" md={4}>Hostname</Col>
          <Col className="node-value" lg={8}>{node.properties.hostname}</Col>
        </Row>
      ) }
      { node.properties.cloud_infos && Object.keys(node.properties.cloud_infos).length > 0 && (<div>
        <Row>
          <Col className="node-key" md={4}>Cloud Provider</Col>
          <Col className="node-value" lg={8}>{node.properties.cloud_infos.provider}</Col>
        </Row>
        <Row>
          <Col className="node-key" md={4}>Instance ID</Col>
          <Col className="node-value" lg={8}>{node.properties.cloud_infos.instance_id}</Col>
        </Row>
        <Row>
          <Col className="node-key" md={4}>Instance Type</Col>
          <Col className="node-value" lg={8}>{node.properties.cloud_infos.instance_type}</Col>
        </Row>
        <Row>
          <Col className="node-key" md={4}>Instance IP</Col>
          <Col className="node-value" lg={8}>{node.properties.cloud_infos.private_ip}</Col>
        </Row>
        <Row>
          <Col className="node-key" md={4}>Availability Zone</Col>
          <Col className="node-value" lg={8}>{node.properties.cloud_infos.availability_zone}</Col>
        </Row>
      </div>) }
      <Row>
        <Col className="node-key" md={4}>Request URI</Col>
        <Col className="node-value" lg={8}>{node.properties.uri}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Request URI</Col>
        <Col className="node-value" lg={8}>{node.properties.uri}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Request URI</Col>
        <Col className="node-value" lg={8}>{node.properties.uri}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>IP Address</Col>
        <Col className="node-value" lg={8}>{ips}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>IP Address</Col>
        <Col className="node-value" lg={8}>{ips}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>IP Address</Col>
        <Col className="node-value" lg={8}>{ips}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Duration</Col>
        <Col className="node-value" lg={8}>{node.properties.duration}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Duration</Col>
        <Col className="node-value" lg={8}>{node.properties.duration}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Duration</Col>
        <Col className="node-value" lg={8}>{node.properties.duration}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Type</Col>
        <Col className="node-value" lg={8}>{node.properties.type}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Type</Col>
        <Col className="node-value" lg={8}>{node.properties.type}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Type</Col>
        <Col className="node-value" lg={8}>{node.properties.type}</Col>
      </Row>
      <Row>
        <Col className="node-key" md={4}>Response</Col>
        <Col className="node-value" lg={8}>{node.properties.response}</Col>
      </Row>
    </Container>
        </Col>
      </Row>
    </Container>
  )
}

class Timeline extends React.Component {

  constructor(props) {
    super(props);

    this.state = {
      url: this.props.url,
      refresh: this.props.refresh,
      loaded: false,
    };
  }

  componentWillMount() {
    this.fetchData(this.state.url);
  }

  componentWillReceiveProps(props) {
    if (props.refresh !== undefined && props.refresh !== this.state.refresh) {
      console.log("Reload data", props.url, props.refresh);
      this.setState({ url: props.url, loaded: false, refresh: props.refresh });
      this.fetchData(props.url);
    }
  }

  fetchData(url) {
    fetch(url)
      .then(res => res.json())
      .then(
        (result) => {
          console.log("response from API:", result);
          var data = processData(result);

          this.setState({ "data": data, loaded: true });
        },
        (error) => {
          console.error("error processing API", error);
        }
      );
  }

  render() {
    if (this.state.loaded === true) {
      return <FlowChartWithState initialValue={this.state.data} Components={{ NodeInner: NodeInnerCustom }} />
    }

    return null
  }
}

export default Timeline
