import * as React from "react";
import SearchBarItem from "./SearchBar";
import Link from "next/link";
import EmptyInstancePage from "./EmptyInstancePage";
import { useRouter } from "next/router";
import { Pagination, Row, Col, Container, Button } from "react-bootstrap";
import { instances } from "../utils/data";
import { GetStaticProps } from "next";
import { instanceService } from "../services/instances.service";
import { Instance } from "../interfaces/Instance";
import { useState, useEffect } from "react";
import useSWR from "swr";
import ErrorPage from "./Error";
import LoadingSpinner from "./LoadingSpinner";
import BottomPagination from "./BottomPagination";

export default function InstancesList() {
  const router = useRouter();
  const fetcher = () => instanceService.readInstances();
  const { data, error } = useSWR("/instances", fetcher);
  const [instances, setInstances] = useState([]);
  const [instanceListShown, setInstanceListShown] = React.useState(instances);
  const [instanceNo, setInstanceNo] = React.useState({
    online: instances?.length,
    total: instances?.length,
  });
  useEffect(() => {
    if (data) {
      setInstances(data.data);
      setInstanceListShown(data.data);
      setInstanceNo({
        online: instances?.length,
        total: instances?.length,
      });
    }
  }, [data]);
  const [searchText, setSearchText] = React.useState("");
  const [currentPage, setCurrentPage] = React.useState(1);
  const setLastPage = () => {
    setCurrentPage(InstancesList.length + 1);
  };
  const handleAddInstance = () => router.push("/instance/add");

  const handleHelp = () => router.push("/help");
  const getShowableInstances = (page: number) => {};
  const handleSearchTextFilter = (value: string) => {
    const filteredGroups = instances.filter(
      (item) => item.name.includes(value) || value.includes(item.name)
    );
    setInstanceListShown(filteredGroups);
  };

  if (error) return <ErrorPage message={error.message} />;
  if (!data) return <LoadingSpinner />;

  return (
    <div>
      <div className="flex flex-wrap justify-between items-center justify-items-stretch space-x-2 space-y-2 m-5">
        <div className="">
          <div className="px-3 py-2 w-max card shadow-md">
            Online Instances {instanceNo.online}/{instanceNo.total}
          </div>
        </div>
        <div className=" col-6">
          <SearchBarItem setSearchText={setSearchText}
            handleOnChange={handleSearchTextFilter}
            />
        </div>
        <div className="btn-group">
          <button
            className="btn btn-md btn-success w-40"
            onClick={handleAddInstance}
          >
            Add New Instance
          </button>
          <button className="btn btn-info" onClick={handleHelp}>
            Help
          </button>
        </div>
      </div>
      {instanceListShown?.length === 0 ? (
        <EmptyInstancePage />
      ) : (
        <div className="px-5 shadow-md">
          <table className="table table-bordered table-hover table-striped">
            <thead>
              <tr>
                <th>SN</th>
                <th>Instance ID</th>
                <th>Instance Name</th>
                <th>IP Address</th>
                <th>Security Group</th>
                <th>Connected on</th>
                <th>Status</th>
                <th>Description</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {instanceListShown?.map((instance, index) => (
                <tr key={index.toString()}>
                  <td>{index + 1}</td>
                  <td>
                    <Link href={`/instance/${instance.id}`}>
                      <a className="btn btn-link">{instance.id}</a>
                    </Link>
                  </td>
                  <td>{instance.name}</td>
                  <td>{instance.ip}</td>
                  <td>
                    <Link
                      href={`security_group/${instance?.securityGroup?.id}`}
                    >
                      {instance.securityGroup.name}
                    </Link>
                  </td>
                  <td>{instance.creationDate}</td>
                  <td>{instance.status? "Live" : "Stopped"} </td>
                  <td>{instance.description}</td>
                  <td>
                    <Link href={`/instance/${instance.id}`}>
                      <a className="btn btn-link">View</a>
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="flex flex-row-reverse">
            <BottomPagination
              currentPage={currentPage}
              setCurrentPage={setCurrentPage}
              setLastPage={setLastPage}
            />
          </div>
        </div>
      )}
    </div>
  );
}
